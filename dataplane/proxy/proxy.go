package proxy

import (
	"encoding/binary"
	"fmt"
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
	conn net.Conn
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
	// connToRealEnd, err := net.DialTimeout("tcp", dst, 2*time.Second)
	// if err != nil {
	// 	logrus.Errorf("failed to connect to %v: %v", dst, err)
	// 	return nil, gnet.Close
	// }

	// connCtx := ConnContext{conn: connToRealEnd}
	// c.SetContext(connCtx)
	return
}

func (p *Proxy) OnTraffic(c gnet.Conn) (action gnet.Action) {
	// TODO 至真实服务器中
	// logrus.Infof("[OnTraffic] - traffic on %s", c.RemoteAddr().String())
	// cc := c.Context()
	// connCtx, ok := cc.(ConnContext)
	// if !ok {
	// 	logrus.Errorf("failed to cast ConnContext to ConnContext")
	// 	return gnet.Close
	// }

	// // TODO 防止协程无限扩张
	// // src -> dst
	// go func() {
	// 	io.Copy(connCtx.conn, c)
	// }()

	// // dst -> src
	// go func() {
	// 	io.Copy(c, connCtx.conn)
	// }()

	return
}

func (p *Proxy) OnClose(c gnet.Conn, _ error) (action gnet.Action) {
	cc := c.Context()
	connCtx, ok := cc.(ConnContext)
	if !ok {
		logrus.Errorf("failed to cast ConnContext to ConnContext")
		return
	}

	connCtx.conn.Close()
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
