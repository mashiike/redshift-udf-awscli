package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/hashicorp/logutils"
	"github.com/mattn/go-shellwords"
	"golang.org/x/sync/errgroup"
)

var (
	Version = "current"
)

var filter = &logutils.LevelFilter{
	Levels:   []logutils.LogLevel{"debug", "info", "warn", "error"},
	MinLevel: logutils.LogLevel("info"),
	Writer:   os.Stderr,
}

type RedshiftUDFInput struct {
	RequestID        string          `json:"request_id,omitempty"`
	Cluster          string          `json:"cluster,omitempty"`
	User             string          `json:"user,omitempty"`
	Database         string          `json:"database,omitempty"`
	ExternalFunction string          `json:"external_function,omitempty"`
	QueryID          int             `json:"query_id,omitempty"`
	NumRecords       int             `json:"num_records,omitempty"`
	Arguments        [][]interface{} `json:"arguments,omitempty"`
}

type RedshiftUDFOutput struct {
	Success    bool          `json:"success"`
	ErrorMsg   string        `json:"error_msg,omitempty"`
	NumRecords int           `json:"num_records,omitempty"`
	Results    []interface{} `json:"results,omitempty"`
}

func main() {
	if os.Getenv("DEBUG") == "true" {
		filter.MinLevel = logutils.LogLevel("debug")
	}
	log.SetOutput(filter)
	log.Printf("[info] msg:launch redshift-udf-awscli\tversion:%s\n", Version)
	cmd := exec.CommandContext(context.Background(), "aws", "--version")
	cmd.Stderr = os.Stderr
	var buf bytes.Buffer
	cmd.Stdout = &buf
	if err := cmd.Run(); err != nil {
		panic(err)
	}
	log.Printf("[info] %s", buf.String())
	lambda.Start(wrap(handler))
}

func handler(ctx context.Context, input *RedshiftUDFInput) (*RedshiftUDFOutput, error) {
	output := &RedshiftUDFOutput{
		Results: make([]interface{}, input.NumRecords),
	}
	var g errgroup.Group
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	for i := 0; i < input.NumRecords; i++ {
		index := i
		args := input.Arguments[i]
		g.Go(func() error {
			if len(args) != 1 {
				return errors.New("argument count not match, expected 1 string argument")
			}
			if args[0] == nil {
				output.Results[index] = nil
				return nil
			}
			commandString, ok := args[0].(string)
			if !ok {
				return errors.New("argument is not string, expected 1 string argument")
			}
			commandString = strings.TrimSpace(commandString)
			log.Printf("[info][request_id=%s][workder=%d] before: %s", input.RequestID, index, commandString)
			if !strings.HasPrefix(commandString, "aws") {
				return errors.New("argument is not aws command string, expected 1 string argument")
			}
			parts, err := shellwords.Parse(commandString)
			if err != nil {
				return fmt.Errorf("command split error: %w", err)
			}
			var endIndex int
			for endIndex = 0; endIndex < len(parts); endIndex++ {
				stmt := strings.TrimSpace(parts[endIndex])
				switch stmt {
				case ";", "&&", "&", "||", "|":
					break
				}
			}
			log.Printf("[info][request_id=%s][workder=%d] after: %s", input.RequestID, index, strings.Join(parts[0:endIndex], " "))
			cmd := exec.CommandContext(ctx, parts[0], parts[1:endIndex]...)
			var bufErr bytes.Buffer
			var bufOut bytes.Buffer
			cmd.Stderr = &bufErr
			cmd.Stdout = &bufOut
			if err := cmd.Run(); err != nil {
				log.Printf("[info][request_id=%s][workder=%d] command stderr: \n%s", input.RequestID, index, bufErr.String())
				return fmt.Errorf("command run failed: %w", err)
			}
			var v interface{}
			if err := json.NewDecoder(&bufOut).Decode(&v); err != nil {
				log.Printf("[info][request_id=%s][workder=%d] command stdout: \n%s", input.RequestID, index, bufOut.String())
				return fmt.Errorf("can not parse as json: %w", err)
			}
			output.Results[index] = v
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}
	output.Success = true
	output.NumRecords = len(output.Results)
	return output, nil
}

func wrap(handler func(ctx context.Context, input *RedshiftUDFInput) (*RedshiftUDFOutput, error)) interface{} {
	return func(ctx context.Context, input *RedshiftUDFInput) (str string, err error) {
		defer func() {
			if panicErr := recover(); panicErr != nil {
				log.Printf("[error] msg:function panic\tdetail:%v\n", panicErr)
				err = errors.New("function panic")
			}
		}()
		log.Printf(
			"[info] request_id:%s\tcluster:%s\tuser:%s\t:database:%s\texternal_function:%s\tquery_id:%d\t:num_records:%d\n",
			input.RequestID,
			input.Cluster,
			input.User,
			input.Database,
			input.ExternalFunction,
			input.QueryID,
			input.NumRecords,
		)
		output, err := handler(ctx, input)
		if err != nil {
			if output == nil {
				output = &RedshiftUDFOutput{}
			}
			output.Success = false
			output.ErrorMsg = err.Error()
		}
		log.Printf("[info] request_id:%s\tquery_id:%d\tsuccess:%t\terror_msg:%s\tnum_records:%d\n",
			input.RequestID,
			input.QueryID,
			output.Success,
			coalesceString(output.ErrorMsg, "-"),
			output.NumRecords,
		)
		var bs []byte
		bs, err = json.Marshal(output)
		str = string(bs)
		return
	}
}

func coalesceString(str1, str2 string) string {
	if str1 == "" {
		return str2
	}
	return str1
}
