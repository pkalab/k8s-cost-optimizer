package pricing

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/costexplorer"
	"github.com/aws/aws-sdk-go-v2/service/costexplorer/types"
)

type Client struct {
	ceClient *costexplorer.Client
}

func New(ctx context.Context) (*Client, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("load AWS config: %w", err)
	}
	return &Client{
		ceClient: costexplorer.NewFromConfig(cfg),
	}, nil
}

type ClusterCost struct {
	CPUPerHour    float64
	MemPerGiBHour float64
}

func (c *Client) GetClusterCost(ctx context.Context, start, end time.Time) (*ClusterCost, error) {
	startStr := start.Format("2006-01-02")
	endStr := end.Format("2006-01-02")
	input := &costexplorer.GetCostAndUsageInput{
		TimePeriod: &types.DateInterval{
			Start: aws.String(startStr),
			End:   aws.String(endStr),
		},
		Granularity: types.GranularityDaily,
		Metrics:     []string{"UnblendedCost"},
		GroupBy: []types.GroupDefinition{
			{Type: types.GroupDefinitionTypeDimension, Key: aws.String("SERVICE")},
		},
	}

	result, err := c.ceClient.GetCostAndUsage(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("get cost and usage: %w", err)
	}

	var totalCost float64
	for _, group := range result.ResultsByTime {
		for _, g := range group.Groups {
			if len(g.Metrics) > 0 {
				if cost, ok := g.Metrics["UnblendedCost"]; ok && cost.Amount != nil {
					if v, err := strconv.ParseFloat(*cost.Amount, 64); err == nil {
						totalCost += v
					}
				}
			}
		}
	}

	days := end.Sub(start).Hours() / 24
	if days < 1 {
		days = 1
	}
	dailyCost := totalCost / days

	return &ClusterCost{
		CPUPerHour:    dailyCost / 24 * 0.5,
		MemPerGiBHour: dailyCost / 24 * 0.5 / 1000,
	}, nil
}
