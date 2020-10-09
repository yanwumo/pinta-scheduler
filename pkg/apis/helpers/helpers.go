package helpers

import (
	"context"
	"fmt"
	pintav1 "github.com/qed-usc/pinta-scheduler/pkg/apis/pintascheduler/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apiserver/pkg/server/healthz"
	"k8s.io/apiserver/pkg/server/mux"
	"k8s.io/klog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var PintaJobKind = pintav1.SchemeGroupVersion.WithKind("PintaJob")

// StartHealthz register healthz interface.
func StartHealthz(healthzBindAddress, name string) error {
	listener, err := net.Listen("tcp", healthzBindAddress)
	if err != nil {
		return fmt.Errorf("failed to create listener: %v", err)
	}

	pathRecorderMux := mux.NewPathRecorderMux(name)
	healthz.InstallHandler(pathRecorderMux)

	server := &http.Server{
		Addr:           listener.Addr().String(),
		Handler:        pathRecorderMux,
		MaxHeaderBytes: 1 << 20,
	}

	return runServer(server, listener)
}

func runServer(server *http.Server, ln net.Listener) error {
	if ln == nil || server == nil {
		return fmt.Errorf("listener and server must not be nil")
	}

	stopCh := make(chan os.Signal)
	signal.Notify(stopCh, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		<-stopCh
		ctx, cancel := context.WithTimeout(context.Background(), 0)
		_ = server.Shutdown(ctx)
		cancel()
	}()

	go func() {
		defer utilruntime.HandleCrash()

		listener := tcpKeepAliveListener{ln.(*net.TCPListener)}

		err := server.Serve(listener)
		msg := fmt.Sprintf("Stopped listening on %s", listener.Addr().String())
		select {
		case <-stopCh:
			klog.Info(msg)
		default:
			klog.Fatalf("%s due to error: %v", msg, err)
		}
	}()

	return nil
}

type tcpKeepAliveListener struct {
	*net.TCPListener
}

// Accept waits for and returns the next connection to the listener.
func (ln tcpKeepAliveListener) Accept() (net.Conn, error) {
	tc, err := ln.AcceptTCP()
	if err != nil {
		return nil, err
	}
	_ = tc.SetKeepAlive(true)
	_ = tc.SetKeepAlivePeriod(3 * time.Minute)
	return tc, nil
}
