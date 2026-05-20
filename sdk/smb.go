package sdk

import (
	"context"
	"fmt"
	"net"
	"strconv"

	"github.com/cloudsoda/go-smb2"
)

func SMBConnect(ctx context.Context, host string, port int, username, password, share string) (net.Conn, *smb2.Session, *smb2.Share, error) {
	if port == 0 {
		port = 445
	}

	addr := net.JoinHostPort(host, strconv.Itoa(port))

	var netDialer net.Dialer
	conn, err := netDialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("dial network failed: %w", err)
	}

	dialer := &smb2.Dialer{
		Initiator: &smb2.NTLMInitiator{
			User:     username,
			Password: password,
		},
	}

	session, err := dialer.DialConn(ctx, conn, addr)
	if err != nil {
		conn.Close()
		return nil, nil, nil, fmt.Errorf("smb authentication failed: %w", err)
	}

	session = session.WithContext(ctx)

	mounted, err := session.Mount(share)
	if err != nil {
		session.Logoff()
		conn.Close()
		return nil, nil, nil, fmt.Errorf("mounting share failed: %w", err)
	}
	return conn, session, mounted, nil
}
