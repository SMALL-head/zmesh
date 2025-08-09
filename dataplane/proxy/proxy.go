package proxy

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/panjf2000/gnet/v2"
	"github.com/sirupsen/logrus"

	"net"
	"syscall"
)

const (
	SO_ORIGINAL_DST = 80
)

type Mode string

const (
	ProxyMode   Mode = "proxy"
	SidecarMode Mode = "sidecar"
)

const (
	outBoundFileName = "o"
	inBoundFileName  = "i"
)

type NoOpLogger struct{}

func (l *NoOpLogger) Debugf(format string, args ...interface{}) {}
func (l *NoOpLogger) Infof(format string, args ...interface{})  {}
func (l *NoOpLogger) Warnf(format string, args ...interface{})  {}
func (l *NoOpLogger) Errorf(format string, args ...interface{}) {}
func (l *NoOpLogger) Fatalf(format string, args ...interface{}) {}

type Proxy struct {
	gnet.EventHandler
	Host     string
	Port     int
	Protocol string
	mode     Mode

	lock sync.Mutex
}

type ProxyOutbound struct {
	*Proxy
}
type ProxyInbound struct {
	*Proxy
}

type Option func(*Proxy)

func WithHost(host string) Option {
	return func(p *Proxy) {
		p.Host = host
	}
}

func WithPort(port int) Option {
	return func(p *Proxy) {
		p.Port = port
	}
}

func WithMode(mode Mode) Option {
	return func(p *Proxy) {
		p.mode = mode
	}
}

func New(opts ...Option) *Proxy {
	p := &Proxy{}
	p.EventHandler = &gnet.BuiltinEventEngine{}
	for _, o := range opts {
		o(p)
	}

	return p
}

func NewProxyOutBound(opts ...Option) *ProxyOutbound {
	p := New(opts...)
	return &ProxyOutbound{Proxy: p}
}

func NewProxyInBound(opts ...Option) *ProxyInbound {
	p := New(opts...)
	return &ProxyInbound{Proxy: p}
}

type ConnContext struct {
	destAddr string
	conn     net.Conn
}

func (p *Proxy) listenAddr() string {
	protocol := p.Protocol
	if protocol == "" {
		protocol = "tcp"
	}
	return fmt.Sprintf("%s://%s:%d", protocol, p.Host, p.Port)
}

func (p *ProxyInbound) Start() error {
	return gnet.Run(p, p.listenAddr(), gnet.WithMulticore(true), gnet.WithLogger(&NoOpLogger{}))
}

func (p *ProxyOutbound) Start() error {
	return gnet.Run(p, p.listenAddr(), gnet.WithMulticore(true), gnet.WithLogger(&NoOpLogger{}))
}

func (p *ProxyOutbound) OnBoot(eng gnet.Engine) (action gnet.Action) {
	logrus.Infof("starting outbound server on %s", p.listenAddr())
	if p.mode != ProxyMode && p.mode != SidecarMode {
		logrus.Errorf("invalid mode: %s, only support %s and %s",
			p.mode, ProxyMode, SidecarMode)
		return gnet.Shutdown
	}

	// 启动一个协程监听系统信号，优雅关闭
	go func() {
		stopCh := make(chan os.Signal, 1)
		signal.Notify(stopCh, syscall.SIGINT, syscall.SIGTERM)
		<-stopCh
		_ = eng.Stop(context.TODO())
	}()

	return
}

func (p *ProxyOutbound) OnOpen(c gnet.Conn) (out []byte, action gnet.Action) {
	logrus.Infof("opening connection on %s", c.RemoteAddr().String())
	switch p.mode {
	case SidecarMode:
		return sidecarModeOpenHandler(c, outBoundFileName)
	case ProxyMode:
		return proxyModeOpenHandler(c, "127.0.0.1:8888") // 该模式用作测试，这里直接写死
	default:
		logrus.Errorf("unsupported mode: %s", p.mode)
		return nil, gnet.Shutdown
	}
}

func (p *ProxyOutbound) OnTraffic(c gnet.Conn) (action gnet.Action) {
	// TODO 至真实服务器中
	cc := c.Context()
	connCtx, ok := cc.(ConnContext)
	if !ok {
		logrus.Errorf("[OutBoundOnTraffic] - failed to cast ConnContext")
		return gnet.Close
	}

	// 按需建立连接
	// 这里使用function包起来的主要目的是为了lock的作用域
	// 锁的竞争不会很激烈，因此这里不使用双重校验锁了
	// connErr := func() gnet.Action {
	// 	p.lock.Lock()
	// 	defer p.lock.Unlock()
	// 	if connCtx.conn == nil {
	// 		d := &net.Dialer{}
	// 		conn, err := d.Dial("tcp", connCtx.destAddr)
	// 		if err != nil {
	// 			logrus.Errorf("failed to connect to %v: %v", connCtx.destAddr, err)
	// 			// 远端异常回传给gnet
	// 			return gnet.Close
	// 		}
	// 		connCtx.conn = conn

	// 		// dst -> src 将实际的数据回传给gnet连接
	// 		go func() {
	// 			_, err = io.Copy(c, connCtx.conn)
	// 			if err != nil {
	// 				logrus.Errorf("failed to copy data from connection to gnet conn: %v", err)
	// 			} else {
	// 				logrus.Infoln("Connection closed normally")
	// 			}
	// 			logrus.Infof("connection to %s closed", connCtx.destAddr)
	// 		}()
	// 		c.SetContext(connCtx)
	// 	}
	// 	return gnet.None
	// }()

	// if connErr == gnet.Close {
	// 	return gnet.Close
	// }

	// 将data送到conn里面
	if connCtx.conn == nil {
		logrus.Errorf("[OutBoundOnTraffic] - connection to %s is nil, cannot send data", connCtx.destAddr)
		return gnet.Close
	}
	dataSize := c.InboundBuffered()
	data, err := c.Next(dataSize)
	if err != nil {
		logrus.Errorf("[OutBoundOnTraffic] - failed to read data from connection: %v", err)
		return gnet.Close
	}
	_, err = connCtx.conn.Write(data)
	if err != nil {
		logrus.Errorf("[OutBoundOnTraffic] - failed to copy data to connection: %v", err)
		return gnet.Close
	}

	return
}

func (p *ProxyOutbound) OnClose(c gnet.Conn, _ error) (action gnet.Action) {
	logrus.Infof("closing connection on %s", c.RemoteAddr().String())
	cc := c.Context()
	connCtx, ok := cc.(ConnContext)
	if !ok {
		logrus.Errorf("[OutBoundOnClose] - failed to cast ConnContext to ConnContext")
		return
	}
	if connCtx.conn != nil {
		connCtx.conn.Close()
	}

	return
}

func (p *ProxyInbound) OnBoot(eng gnet.Engine) (action gnet.Action) {
	logrus.Infof("starting inbound server on %s", p.listenAddr())
	if p.mode != ProxyMode && p.mode != SidecarMode {
		logrus.Errorf("invalid mode: %s, only support %s and %s",
			p.mode, ProxyMode, SidecarMode)
		return gnet.Shutdown
	}

	// 启动一个协程监听系统信号，优雅关闭
	go func() {
		stopCh := make(chan os.Signal, 1)
		signal.Notify(stopCh, syscall.SIGINT, syscall.SIGTERM)
		<-stopCh
		_ = eng.Stop(context.TODO())
	}()

	return
}

func (p *ProxyInbound) OnOpen(c gnet.Conn) (out []byte, action gnet.Action) {
	logrus.Infof("[InBoundOnOpen] - opening connection from %s", c.RemoteAddr().String())
	switch p.mode {
	case SidecarMode:
		return sidecarModeOpenHandler(c, inBoundFileName)
	case ProxyMode:
		return proxyModeOpenHandler(c, "127.0.0.1:8888")
	default:
		logrus.Errorf("[InBoundOnOpen] - unsupported mode: %s", p.mode)
		return nil, gnet.Close
	}
}

func (p *ProxyInbound) OnTraffic(c gnet.Conn) (action gnet.Action) {
	cc := c.Context()
	connCtx, ok := cc.(ConnContext)
	if !ok {
		logrus.Errorf("[InBoundOnTraffic] - failed to cast ConnContext")
		return gnet.Close
	}
	if connCtx.conn == nil {
		logrus.Errorf("[InBoundOnTraffic] - connection to %s is nil, cannot send data", connCtx.destAddr)
		return gnet.Close
	}
	dataSize := c.InboundBuffered()
	data, err := c.Next(dataSize)
	if err != nil {
		logrus.Errorf("[InBoundOnTraffic] - failed to read data from connection: %v", err)
		return gnet.Close
	}

	_, err = connCtx.conn.Write(data)
	if err != nil {
		logrus.Errorf("[InBoundOnTraffic] - failed to copy data to connection: %v", err)
		return gnet.Close
	}
	return
}

func getOriginDst(fd int) (originDst string, host string, port uint16, err error) {

	// 下面的注释来自:https://gist.github.com/fangdingjun/11e5d63abe9284dc0255a574a76bbcb1
	// Get original destination
	// this is the only syscall in the Golang libs that I can find that returns 16 bytes
	// Example result: &{Multiaddr:[2 0 31 144 206 190 36 45 0 0 0 0 0 0 0 0] Interface:0}
	// port starts at the 3rd byte and is 2 bytes long (31 144 = port 8080)
	// IPv6 version, didn't find a way to detect network family
	//addr, err := syscall.GetsockoptIPv6Mreq(int(clientConnFile.Fd()), syscall.IPPROTO_IPV6, IP6T_SO_ORIGINAL_DST)
	// IPv4 address starts at the 5th byte, 4 bytes long (206 190 36 45)
	var addr *syscall.IPv6Mreq
	addr, err = syscall.GetsockoptIPv6Mreq(fd, syscall.IPPROTO_IP, SO_ORIGINAL_DST)
	if err != nil {
		//logrus.Errorf("[getOriginDst] - error getting SO_ORIGINAL_DST: %s", err)
		return "", "", 0, err
	}

	port = binary.BigEndian.Uint16(addr.Multiaddr[2:4])
	host = fmt.Sprintf("%d.%d.%d.%d", addr.Multiaddr[4], addr.Multiaddr[5], addr.Multiaddr[6], addr.Multiaddr[7])

	originDst = fmt.Sprintf("%s:%d", host, port)

	return originDst, host, port, nil
}

// fileName用于表示基于 c gnet.Conn 打开的文件唯一标识
func sidecarModeOpenHandler(c gnet.Conn, fileName string) (out []byte, action gnet.Action) {
	rawConnFd := c.Fd()
	dst, _, _, err := getOriginDst(rawConnFd)
	if err != nil {
		logrus.Errorf("failed to get origin dst %v", err)
		return nil, gnet.Close
	}
	if dst == "" {
		logrus.Errorf("origin dst is empty")
		return nil, gnet.Close
	}
	logrus.Infof("[OnOpen]: origin dst: %s", dst)
	// 设置连接上下文d := &net.Dialer{}

	connCtx := ConnContext{destAddr: dst}
	d := net.Dialer{
		Timeout: 5 * time.Second,
	}
	conn, err := d.Dial("tcp", connCtx.destAddr)
	if err != nil {
		logrus.Errorf("failed to connect to %v: %v", connCtx.destAddr, err)
		// 远端异常回传给gnet
		return nil, gnet.Close
	}
	connCtx.conn = conn
	go func() {
		// dst -> src 将实际的数据回传给gnet连接
		fd := c.Fd()
		f := os.NewFile(uintptr(fd), fileName)
		if f == nil {
			logrus.Errorf("failed to create os.File from fd %d", fd)
			return
		}

		_, err = io.Copy(f, connCtx.conn)
		if err != nil {
			logrus.Errorf("failed to copy data from connection to gnet conn: %v", err)
		} else {
			logrus.Infoln("Connection closed normally")
		}
		logrus.Infof("connection to %s closed", connCtx.destAddr)
		connCtx.conn.Close()
		connCtx.conn = nil
	}()
	c.SetContext(connCtx)
	return
}

func proxyModeOpenHandler(c gnet.Conn, dst string) (out []byte, action gnet.Action) {
	logrus.Infof("[OnOpen] - [proxyModeOpenHandler] - origin dst: %s", dst)
	connCtx := ConnContext{destAddr: dst}
	d := net.Dialer{}
	conn, err := d.Dial("tcp", connCtx.destAddr)
	if err != nil {
		logrus.Errorf("[OnOpen] - [proxyModeOpenHandler] - failed to connect to %v: %v", connCtx.destAddr, err)
		return nil, gnet.Close
	}

	connCtx.conn = conn
	go func() {
		fd := c.Fd()
		f := os.NewFile(uintptr(fd), "real-end")
		if f == nil {
			logrus.Errorf("[OnOpen] - [proxyModeOpenHandler] - failed to create os.File from fd %d", fd)
			return
		}
		_, err = io.Copy(f, connCtx.conn)
		if err != nil {
			logrus.Errorf("[OnOpen] - [proxyModeOpenHandler] - failed to copy data from connection to gnet conn: %v", err)
		} else {
			logrus.Infoln("[OnOpen] - [proxyModeOpenHandler] - Connection closed normally")
		}
		logrus.Infof("[OnOpen] - [proxyModeOpenHandler] - connection to %s closed", connCtx.destAddr)
		connCtx.conn.Close()
		connCtx.conn = nil
	}()

	c.SetContext(connCtx)
	return
}
