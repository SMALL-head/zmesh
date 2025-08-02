package proxy

import (
	"encoding/binary"
	"fmt"
	"io"
	"sync"

	"github.com/panjf2000/gnet/v2"
	"github.com/sirupsen/logrus"

	"net"
	"syscall"
)

const (
	SO_ORIGINAL_DST = 80
)

type Proxy struct {
	gnet.EventHandler
	Host     string
	Port     int
	Protocol string

	lock sync.Locker
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

func New(opts ...Option) *Proxy {
	p := &Proxy{}
	p.EventHandler = &gnet.BuiltinEventEngine{}
	for _, o := range opts {
		o(p)
	}

	return p
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

func (p *Proxy) Start() error {
	return gnet.Run(p, p.listenAddr(), gnet.WithMulticore(true))
}

func (p *Proxy) OnBoot(eng gnet.Engine) (action gnet.Action) {
	logrus.Infof("starting server on %s", p.listenAddr())
	return
}

func (p *Proxy) OnOpen(c gnet.Conn) (out []byte, action gnet.Action) {
	logrus.Infof("opening connection on %s", c.RemoteAddr().String())
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

	connCtx := ConnContext{destAddr: dst}
	c.SetContext(connCtx)
	return
}

func (p *Proxy) OnTraffic(c gnet.Conn) (action gnet.Action) {
	// TODO 至真实服务器中
	cc := c.Context()
	connCtx, ok := cc.(ConnContext)
	if !ok {
		logrus.Errorf("failed to cast ConnContext")
		return gnet.Close
	}

	// 按需建立连接
	// 这里使用function包起来的主要目的是为了lock的作用域
	// 锁的竞争不会很激烈，因此这里不使用双重校验锁了
	connErr := func() gnet.Action {
		p.lock.Lock()
		defer p.lock.Unlock()
		if connCtx.conn == nil {
			d := &net.Dialer{}
			conn, err := d.Dial("tcp", connCtx.destAddr)
			if err != nil {
				logrus.Errorf("failed to connect to %v: %v", connCtx.destAddr, err)
				// 远端异常回传给gnet
				return gnet.Close
			}
			connCtx.conn = conn

			// dst -> src 将实际的数据回传给gnet连接
			go func() {
				_, err = io.Copy(c, connCtx.conn)
				if err != nil {
					logrus.Errorf("failed to copy data from connection to gnet conn: %v", err)
				} else {
					logrus.Infoln("Connection closed normally")
				}
				logrus.Infof("connection to %s closed", connCtx.destAddr)
			}()
			c.SetContext(connCtx)
		}
		return gnet.None
	}()

	if connErr == gnet.Close {
		return gnet.Close
	}

	// 将data送到conn里面

	dataSize := c.InboundBuffered()
	data, err := c.Next(dataSize)
	if err != nil {
		logrus.Errorf("failed to read data from connection: %v", err)
		return gnet.Close
	}
	_, err = connCtx.conn.Write(data)
	if err != nil {
		logrus.Errorf("failed to copy data to connection: %v", err)
		return gnet.Close
	}

	return
}

func (p *Proxy) OnClose(c gnet.Conn, _ error) (action gnet.Action) {
	logrus.Infof("closing connection on %s", c.RemoteAddr().String())
	cc := c.Context()
	connCtx, ok := cc.(ConnContext)
	if !ok {
		logrus.Errorf("failed to cast ConnContext to ConnContext")
		return
	}
	if connCtx.conn != nil {
		connCtx.conn.Close()
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
