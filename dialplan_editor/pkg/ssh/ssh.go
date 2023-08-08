package ssh

import (
	"context"
	"fmt"
	"myapp/config"
	"myapp/pkg/jaegerotel"
	"time"

	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel/codes"
	"golang.org/x/crypto/ssh"
)

func New(tctx context.Context, cfg config.Config) (*ssh.Client, error) {
	_, spanSsh := jaegerotel.StartSpan(tctx, "SSH - connect")

	sshConfig := ssh.ClientConfig{
		User:            cfg.SshUser,
		Timeout:         180 * time.Second,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Auth: []ssh.AuthMethod{
			ssh.Password(cfg.SshPassword),
		},
	}

	conn, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", cfg.SshHost, cfg.SshPort), &sshConfig)
	if err != nil {
		spanSsh.SetStatus(codes.Error, err.Error())
		spanSsh.End()
		return nil, err
	}

	log.Print("Подключение по ssh успешно")

	spanSsh.End()
	return conn, nil
}
