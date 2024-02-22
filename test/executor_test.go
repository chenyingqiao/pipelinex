package test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/chenyingqiao/pipelinex/executor/kubenetes"
)

func TestKubeExecutor(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*1800)
	defer cancel()
	kubenetes.Boot(ctx, kubenetes.StartupParam{
		MasterUrl:  "",
		Kubeconfig: "/Users/lerko/.kube/config",
	})
	for {
		select {
		case <-ctx.Done():
			fmt.Println("程序结束")
		}
	}
}
