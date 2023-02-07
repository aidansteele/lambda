package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/lambda/types"
	"github.com/fatih/color"
	"github.com/mattn/go-isatty"
	flag "github.com/spf13/pflag"
	"io"
	"os"
	"strings"
	"time"
)

func main() {
	ctx := context.Background()
	err := ctxmain(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}

func logln(msg string) {
	color.New(color.Bold).Fprintln(color.Error, msg)
}

func ctxmain(ctx context.Context) error {
	var err error
	var functionName, inputPath, outputPath string
	flag.StringVarP(&functionName, "function", "f", "", "lambda function name or arn")
	flag.StringVarP(&inputPath, "input", "i", "-", "input file path. default - is stdin")
	flag.StringVarP(&outputPath, "output", "o", "-", "output file path. default - is stdout")
	flag.Parse()

	if functionName == "" {
		return fmt.Errorf("function name must be specified")
	}

	input := os.Stdin
	if inputPath != "-" {
		input, err = os.Open(inputPath)
		if err != nil {
			return fmt.Errorf("opening input file: %w", err)
		}
	}

	output := os.Stdout
	if outputPath != "-" {
		output, err = os.Create(outputPath)
		if err != nil {
			return fmt.Errorf("opening output file: %w", err)
		}
	}

	opts := []func(options *config.LoadOptions) error{
		//config.WithClientLogMode(aws.LogRequestWithBody|aws.LogResponseWithBody),
	}
	cfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	if input == os.Stdin && isatty.IsTerminal(os.Stdin.Fd()) {
		logln("Reading input from stdin. Press Ctrl+D when input is complete.")
	}

	inputPayload, err := io.ReadAll(input)
	if err != nil {
		return fmt.Errorf("reading input: %w", err)
	}

	start := time.Now()
	logln(fmt.Sprintf("Invoking Lambda function %s now (%s)", functionName, start.Format(time.Stamp)))

	api := lambda.NewFromConfig(cfg)
	invoke, err := api.Invoke(ctx, &lambda.InvokeInput{
		FunctionName: &functionName,
		//Qualifier:    nil,
		Payload: inputPayload,
		LogType: types.LogTypeTail,
	})
	if err != nil {
		return fmt.Errorf(": %w", err)
	}

	logTailBytes, err := base64.StdEncoding.DecodeString(*invoke.LogResult)
	if err != nil {
		return fmt.Errorf("err base64 decoding log tail: %w", err)
	}

	logTail := string(logTailBytes)
	startIndex := strings.Index(logTail, "START RequestId")
	if startIndex >= 0 {
		// TODO: don't throw away INIT_START and extension stuff
		logln("Function logs:")
		fmt.Fprintln(os.Stderr, logTail[startIndex:])

		if output == os.Stdout && isatty.IsTerminal(os.Stdout.Fd()) {
			logln("Function output:")
		}
		_, err = fmt.Fprintln(output, string(invoke.Payload))
		if err != nil {
			return fmt.Errorf("writing output: %w", err)
		}

		ferr := aws.ToString(invoke.FunctionError)
		if ferr != "" {
			logln("Function error:")
			fmt.Fprintln(os.Stderr, ferr)
			os.Exit(2)
		} else {
			os.Exit(0)
		}
	} else {
		return fmt.Errorf("fetching of 4KB+ logs not implemented yet")
	}

	//requestId, ok := middleware.GetRequestIDMetadata(invoke.ResultMetadata)
	//if !ok {
	//	return fmt.Errorf("response didn't include a request id")
	//}
	//
	//endTime, ok := middleware.GetServerTime(invoke.ResultMetadata)
	//if !ok {
	//	return fmt.Errorf("response didn't include a Date header")
	//}
	//
	//durationMillis := time.Now().Sub(start).Milliseconds()
	//endTimeMillis := endTime.Add(time.Second).UnixNano() / 1e6
	//startTimeMillis := endTimeMillis - durationMillis
	//
	//logs := cloudwatchlogs.NewFromConfig(cfg)
	//p := cloudwatchlogs.NewFilterLogEventsPaginator(logs, &cloudwatchlogs.FilterLogEventsInput{
	//	FilterPattern: aws.String(fmt.Sprintf(`"RequestId" "%s"`, requestId)),
	//	LogGroupName:  aws.String(fmt.Sprintf("/aws/lambda/%s", functionName)),
	//	StartTime:     &startTimeMillis,
	//	EndTime:       &endTimeMillis,
	//})
	//
	//for p.HasMorePages() {
	//	page, err := p.NextPage(ctx)
	//	if err != nil {
	//		return fmt.Errorf("searching logs: %w", err)
	//	}
	//
	//	for _, event := range page.Events {
	//		msg := *event.Message
	//		if strings.HasPrefix(msg, "START RequestId") {
	//
	//		} else if strings.HasPrefix(msg, "REPORT RequestId") {
	//
	//		}
	//	}
	//}
	//
	//ferr := aws.ToString(invoke.FunctionError)
	//if ferr != "" {
	//	return fmt.Errorf("lambda invocation error: %s", ferr)
	//}

	return nil
}
