package metrics

import (
	"context"
	"log"
	"sync/atomic"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	aCfg "github.com/easc01/websocket-app/pkg/config"
)

var cwClient *cloudwatch.Client

func init() {
	cfg, err := config.LoadDefaultConfig(context.Background(), config.WithRegion("ap-south-1"))
	if err != nil {
		panic("unable to load AWS config: " + err.Error())
	}
	cwClient = cloudwatch.NewFromConfig(cfg)
	log.Println("connected to cloudwatch")
}

var (
	activeConnectionsCount     uint64
	messagesDeliveredCount     uint64
	internalMessageCount       uint64
	unexpectedDisconnectsCount uint64

	latencyTotal uint64
	latencyCount uint64
)

func OnClientConnect()        { atomic.AddUint64(&activeConnectionsCount, 1) }
func OnClientDisconnect()     { atomic.AddUint64(&activeConnectionsCount, ^uint64(0)) }
func OnMessageDelivered()     { atomic.AddUint64(&messagesDeliveredCount, 1) }
func OnUnexpectedDisconnect() { atomic.AddUint64(&unexpectedDisconnectsCount, 1) }
func OnMessageReceived()      { atomic.AddUint64(&internalMessageCount, 1) }

func OnLatencyReport(latencyMs float64) {
	atomic.AddUint64(&latencyTotal, uint64(latencyMs))
	atomic.AddUint64(&latencyCount, 1)
}

func StartCloudWatchPusher(interval time.Duration) chan struct{} {
	stop := make(chan struct{})
	ticker := time.NewTicker(interval)

	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				pushMetricsToCloudWatch()
			case <-stop:
				return
			}
		}
	}()

	return stop
}

func pushMetricsToCloudWatch() {
	active := atomic.LoadUint64(&activeConnectionsCount)
	msgTotal := atomic.LoadUint64(&internalMessageCount)
	msgDelivered := atomic.LoadUint64(&messagesDeliveredCount)
	unexpected := atomic.LoadUint64(&unexpectedDisconnectsCount)

	// Compute average latency
	totalLatency := atomic.SwapUint64(&latencyTotal, 0)
	countLatency := atomic.SwapUint64(&latencyCount, 0)

	var avgLatency float64
	if countLatency > 0 {
		avgLatency = float64(totalLatency) / float64(countLatency)
	}

	// Push metrics to CloudWatch
	_, err := cwClient.PutMetricData(context.Background(), &cloudwatch.PutMetricDataInput{
		Namespace: aws.String("wss/metrics"),
		MetricData: []types.MetricDatum{
			{
				MetricName: aws.String("ActiveConnections"),
				Value:      aws.Float64(float64(active)),
				Unit:       types.StandardUnitCount,
				Dimensions: []types.Dimension{
					{
						Name:  aws.String("ServerID"),
						Value: aws.String(aCfg.AppConfig.ServerID),
					},
				},
			},
			{
				MetricName: aws.String("MessagesTotal"),
				Value:      aws.Float64(float64(msgTotal)),
				Unit:       types.StandardUnitCount,
				Dimensions: []types.Dimension{
					{
						Name:  aws.String("ServerID"),
						Value: aws.String(aCfg.AppConfig.ServerID),
					},
				},
			},
			{
				MetricName: aws.String("MessagesDelivered"),
				Value:      aws.Float64(float64(msgDelivered)),
				Unit:       types.StandardUnitCount,
				Dimensions: []types.Dimension{
					{
						Name:  aws.String("ServerID"),
						Value: aws.String(aCfg.AppConfig.ServerID),
					},
				},
			},
			{
				MetricName: aws.String("UnexpectedDisconnects"),
				Value:      aws.Float64(float64(unexpected)),
				Unit:       types.StandardUnitCount,
				Dimensions: []types.Dimension{
					{
						Name:  aws.String("ServerID"),
						Value: aws.String(aCfg.AppConfig.ServerID),
					},
				},
			},
			{
				MetricName: aws.String("AverageLatencyMs"),
				Value:      aws.Float64(avgLatency),
				Unit:       types.StandardUnitMilliseconds,
				Dimensions: []types.Dimension{
					{
						Name:  aws.String("ServerID"),
						Value: aws.String(aCfg.AppConfig.ServerID),
					},
				},
			},
		},
	})
	if err != nil {
		log.Printf("failed to push metrics to CloudWatch: %v", err)
	} else {
		log.Println("pushed metrics to cloudwatch")
	}
}
