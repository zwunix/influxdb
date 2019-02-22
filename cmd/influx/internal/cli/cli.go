package cli

import (
	"context"
	"io"
	"os"
)

type CLI struct {
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer

	ctx    context.Context
	cancel context.CancelFunc

	client *client
}

func NewCLI(c Config) *CLI {
	ctx, cancel := context.WithCancel(context.Background())
	return &CLI{
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
		ctx:    ctx,
		cancel: cancel,
		client: NewClient(c),
	}
}

func (c *CLI) Open() error {
	if err := c.client.Open(c.ctx); err != nil {
		return err
	}

	return nil
}

func (c *CLI) Close() error {
	if err := c.client.Close(c.ctx); err != nil {
		return err
	}

	return nil
}

//type Request struct {
//	flags map[string]string
//	args  []string
//}
//
//func (c *CLI) executeFindBuckets(flags map[string]string, args []string) (interface{}, error) {
//	bs, _, err := c.client.FindBuckets(c.ctx, nil)
//	if err != nil {
//		return nil, err
//	}
//
//	return bs, nil
//}
//
//func (c *CLI) Exec(fn func() (interface{}, error)) {
//	v, err := fn()
//	if err != nil {
//		// transform err into bytes
//		return
//	}
//
//	// encode result
//}
//
//var commands = map[string]func() (interface{}, error){
//	"create user": nil,
//	"user create": nil,
//}

//func Register("influx user create", func(){})
//func Register(command string, fn func()) {}
//
//func init() {
//	cmd := NewCommand("influx")
//	cmd.Flags("local", "", false)
//	cmd.Flags("token", "t", "footoken")
//
//	userCmd := cmd.NewCommand("influx user")
//	userCmd.Flags("user", "u", "")
//	userCmd.Flags("user-id", "", "")
//
//	userFindCmd := NewCommand("influx user find")
//
//	thing.Register("influx user create", cmd)
//}
