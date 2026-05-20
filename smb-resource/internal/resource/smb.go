package resource

import (
	"context"
	"fmt"
	"net"
	"strconv"

	"github.com/cloudsoda/go-smb2"
)

func (d *Driver) connect(ctx context.Context, source Source) (net.Conn, *smb2.Session, *smb2.Share, error) {
	port := source.Port
	if port == 0 {
		port = 445
	}

	addr := net.JoinHostPort(source.Host, strconv.Itoa(port))

	var netDialer net.Dialer
	conn, err := netDialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("dial network failed: %w", err)
	}

	dialer := &smb2.Dialer{
		Initiator: &smb2.NTLMInitiator{
			User:     source.Username,
			Password: source.Password,
		},
	}

	session, err := dialer.DialConn(ctx, conn, addr)
	if err != nil {
		conn.Close()
		return nil, nil, nil, fmt.Errorf("smb authentication failed: %w", err)
	}
	
	// attach the context to the session so that it can be used in subsequent calls
	session = session.WithContext(ctx)

	share, err := session.Mount(source.Share)
	if err != nil {
		session.Logoff()
		conn.Close()
		return nil, nil, nil, fmt.Errorf("mounting share failed: %w", err)
	}
	return conn, session, share, nil
}
