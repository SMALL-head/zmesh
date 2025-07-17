package proxy_test

import (
	"context"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"io"
	"net"
	"os"
	"testing"
	"time"
)

func TestForE2E(t *testing.T) {
	listener, err := net.Listen("tcp", ":8081")
	require.NoError(t, err)
	defer listener.Close()
	go func() {
		serverConn, err := listener.Accept()
		require.NoError(t, err)
		io.Copy(os.Stdout, serverConn)
		logrus.Infoln("server end")
	}()

	go func() {
		clientConn, err2 := net.Dial("tcp", ":8081")
		require.NoError(t, err2)
		defer clientConn.Close()

		clientConn.Write([]byte("hello world\n"))

		time.Sleep(time.Second * 2)

		logrus.Infoln("[client] - after 2 seconds then send another msg")

		clientConn.Write([]byte("hello world2\n"))

	}()

	timeoutCtx, cancelFunc := context.WithTimeout(context.Background(), time.Second*5)

	select {
	case <-timeoutCtx.Done():
		logrus.Infoln("all done")
		cancelFunc()
	}
}
