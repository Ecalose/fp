package fp

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"os"

	"github.com/gospider007/gtls"
)

type Option struct {
	Addr        string
	CertFile    string
	KeyFile     string
	Certificate tls.Certificate
	NextProtos  []string
	Handler     http.Handler
	DomainNames []string
}

func newTlsConfig(option Option) (*tls.Config, error) {
	var tlsConfig *tls.Config
	if option.Certificate.Certificate != nil {
		tlsConfig = &tls.Config{Certificates: []tls.Certificate{option.Certificate}}
	} else if option.CertFile != "" && option.KeyFile != "" {
		if certData, err := os.ReadFile(option.CertFile); err != nil {
			return tlsConfig, err
		} else if cert, err := gtls.LoadCert(certData); err != nil {
			return tlsConfig, err
		} else if keyData, err := os.ReadFile(option.KeyFile); err != nil {
			return tlsConfig, err
		} else if key, err := gtls.LoadCertKey(keyData); err != nil {
			return tlsConfig, err
		} else if certificate, err := gtls.MergeCert(cert, key); err != nil {
			return tlsConfig, err
		} else {
			tlsConfig = &tls.Config{Certificates: []tls.Certificate{certificate}}
		}
	} else if option.DomainNames != nil {
		return gtls.TLS(option.DomainNames)
	} else if certificate, err := gtls.CreateCertWithAddr(net.IPv4(127, 0, 0, 1)); err != nil {
		return tlsConfig, err
	} else {
		tlsConfig = &tls.Config{Certificates: []tls.Certificate{certificate}}
	}
	if tlsConfig.NextProtos == nil {
		if option.NextProtos == nil {
			tlsConfig.NextProtos = []string{"h2", "http/1.1"}
		} else {
			tlsConfig.NextProtos = option.NextProtos
		}
	}
	tlsConfig.InsecureSkipVerify = true
	return tlsConfig, nil
}

func Server(ctx context.Context, handler http.Handler, options ...Option) (err error) {
	var option Option
	if len(options) > 0 {
		option = options[0]
	}
	if option.Addr == "" {
		option.Addr = ":0"
	}
	if ctx == nil {
		ctx = context.TODO()
	}
	server := &server{
		handler: handler,
		connPip: make(chan net.Conn),
	}
	server.ctx, server.cnl = context.WithCancel(ctx)
	if server.listener, err = net.Listen("tcp", option.Addr); err != nil {
		return err
	}
	if server.tlsConfig, err = newTlsConfig(option); err != nil {
		return err
	}
	go server.listen()
	return server.serve()
}
