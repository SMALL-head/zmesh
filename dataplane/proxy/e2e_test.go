package proxy_test

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

type StdOutDecorator struct {
	file               *os.File
	decoratorFormatter string
}

func (d *StdOutDecorator) Write(p []byte) (n int, err error) {
	if d == nil {
		return 0, nil
	}

	_, err = fmt.Fprintf(d.file, d.decoratorFormatter, fmt.Sprintf("%q", string(p)))
	n = len(p) // 为了适配原来的逻辑，这里返回的大小应该和原来相同 ！！
	return
}

// func (d *StdOutDecorator) ReadFrom(p []byte) (n int, err error) {
// 	if d == nil {
// 		return 0, nil
// 	}
// 	return fmt.Fprintf(d.File, d.decoratorFormatter, string(p))

// }

func DecorateStdOut(format string) *StdOutDecorator {
	return &StdOutDecorator{
		file:               os.Stdout,
		decoratorFormatter: format,
	}
}

func TestForE2E(t *testing.T) {
	listener, err := net.Listen("tcp", ":8081")
	require.NoError(t, err)
	defer listener.Close()
	go func() {
		serverConn, err := listener.Accept()
		require.NoError(t, err)

		// 实时复制数据到输出和缓冲区, 该方法外面似乎没有必要加for循环
		_, err = io.Copy(DecorateStdOut("[server] - reveived: %s\n"), serverConn)
		if err != nil && err != io.EOF {
			logrus.Errorf("io.Copy error: %v", err)
		} else {
			// 从io.Copy的源码上来说，如果返回EOF了，此时的err为nil。
			logrus.Infoln("Connection closed normally")
		}

		logrus.Infoln("server end")
		serverConn.Close()
	}()

	go func() {
		clientConn, err2 := net.Dial("tcp", ":8081")

		go func() {
			// 实时复制数据到输出和缓冲区, 该方法外面似乎没有必要加for循环
			_, err = io.Copy(DecorateStdOut("[client] - reveived: %s\n"), clientConn)
			logrus.Infoln("[clinet] - clientConn closed")
		}()
		require.NoError(t, err2)
		defer clientConn.Close()

		clientConn.Write([]byte("hello world\n"))

		time.Sleep(time.Second * 2)

		logrus.Infoln("[client] - after 2 seconds then send another msg")

		clientConn.Write([]byte("hello world2\n"))

		logrus.Infoln("[client] - sleep 5s and close connection")
		time.Sleep(time.Second * 5)
		logrus.Infoln("[client] - close connection")
	}()

	timeoutCtx, cancelFunc := context.WithTimeout(context.Background(), time.Second*10)

	select {
	case <-timeoutCtx.Done():
		logrus.Infoln("all done")
		cancelFunc()
	}
}
