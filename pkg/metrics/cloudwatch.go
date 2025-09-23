package metrics

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
)

var cwClient *cloudwatch.Client

func init() {
	// Load AWS config & create client
	cfg, err := config.LoadDefaultConfig(context.Background(), config.WithRegion("ap-south-1"))
	if err != nil {
		panic("unable to load AWS config: " + err.Error())
	}
	cwClient = cloudwatch.NewFromConfig(cfg)
}

// StartCloudWatchPusher pushes metrics every interval
func StartCloudWatchPusher(interval time.Duration) {
	go func() {
		for range time.Tick(interval) {
			pushMetricsToCloudWatch()
		}
	}()
}

func pushMetricsToCloudWatch() {
	active := atomic.LoadUint64(&activeConnectionsCount)
	msgTotal := atomic.LoadUint64(&internalMessageCount)
	msgDelivered := atomic.LoadUint64(&messagesDeliveredCount)
	unexpected := atomic.LoadUint64(&unexpectedDisconnectsCount)

	_, _ = cwClient.PutMetricData(context.Background(), &cloudwatch.PutMetricDataInput{
		Namespace: aws.String("WSS"),
		MetricData: []types.MetricDatum{
			{
				MetricName: aws.String("ActiveConnections"),
				Value:      aws.Float64(float64(active)),
				Unit:       types.StandardUnitCount,
			},
			{
				MetricName: aws.String("MessagesTotal"),
				Value:      aws.Float64(float64(msgTotal)),
				Unit:       types.StandardUnitCount,
			},
			{
				MetricName: aws.String("MessagesDelivered"),
				Value:      aws.Float64(float64(msgDelivered)),
				Unit:       types.StandardUnitCount,
			},
			{
				MetricName: aws.String("UnexpectedDisconnects"),
				Value:      aws.Float64(float64(unexpected)),
				Unit:       types.StandardUnitCount,
			},
		},
	})
}

// Replace Prometheus counters with atomic uint64 variables
var (
	activeConnectionsCount     uint64
	messagesDeliveredCount     uint64
	unexpectedDisconnectsCount uint64
	internalMessageCount       uint64
)

// Call these in WebSocket handlers
func OnClientConnect()        { atomic.AddUint64(&activeConnectionsCount, 1) }
func OnClientDisconnect()     { atomic.AddUint64(&activeConnectionsCount, ^uint64(0)) }
func OnMessageDelivered()     { atomic.AddUint64(&messagesDeliveredCount, 1) }
func OnUnexpectedDisconnect() { atomic.AddUint64(&unexpectedDisconnectsCount, 1) }
func OnMessageReceived()      { atomic.AddUint64(&internalMessageCount, 1) }
