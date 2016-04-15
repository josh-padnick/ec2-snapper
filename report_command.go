package main

import (
	"github.com/mitchellh/cli"
	"flag"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
)

const DEFAULT_METRIC_VALUE = 1
const DEFAULT_METRIC_UNIT = cloudwatch.StandardUnitCount

type ReportCommand struct {
	Ui 			cli.Ui
	AwsRegion 		string
	Namespace 		string
	MetricName 		string
	MetricValue 		float64
	MetricUnit 		string
}

// descriptions for args
var reportDscrAwsRegion = "The AWS region to use (e.g. us-west-2)"
var reportDscrNamespace = "The CloudWatch namespace for this metric (e.g. MyCustomMetrics)."
var reportDscrMetricName = "The name of the metric (e.g. MyEC2Backup)."
var reportDscrMetricValue = fmt.Sprintf("The value of the metric (e.g. 1). Defaults to %d.", DEFAULT_METRIC_VALUE)
var reportDscrMetricUnit = fmt.Sprintf("The unit of the metric (e.g. Count). Defaults to %s.", DEFAULT_METRIC_UNIT)

func (c *ReportCommand) Help() string {
	return `ec2-snapper report <args> [--help]

Report a metric to CloudWatch.

Available args are:
--region      		` + reportDscrAwsRegion + `
--namespace      	` + reportDscrNamespace + `
--name      		` + reportDscrMetricName + `
--value    		` + reportDscrMetricValue + `
--unit    		` + reportDscrMetricUnit
}

func (c *ReportCommand) Synopsis() string {
	return "Report a metric to CloudWatch"
}

func (c *ReportCommand) Run(args []string) int {

	// Handle the command-line args
	cmdFlags := flag.NewFlagSet("report", flag.ExitOnError)
	cmdFlags.Usage = func() {
		c.Ui.Output(c.Help())
	}

	cmdFlags.StringVar(&c.AwsRegion, "region", "", reportDscrAwsRegion)
	cmdFlags.StringVar(&c.Namespace, "namespace", "", reportDscrNamespace)
	cmdFlags.StringVar(&c.MetricName, "name", "", reportDscrMetricName)
	cmdFlags.Float64Var(&c.MetricValue, "value", DEFAULT_METRIC_VALUE, reportDscrMetricValue)
	cmdFlags.StringVar(&c.MetricUnit, "unit", DEFAULT_METRIC_UNIT, reportDscrMetricUnit)

	if err := cmdFlags.Parse(args); err != nil {
		return 1
	}

	if err := report(*c); err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	return 0
}

func report(c ReportCommand) error {
	if err := validateReportArgs(c); err != nil {
		return err
	}

	session := session.New(&aws.Config{Region: &c.AwsRegion})
	svc := cloudwatch.New(session)

	return createMetric(c, svc)
}

func createMetric(c ReportCommand, svc *cloudwatch.CloudWatch) error {

	metricData := &cloudwatch.MetricDatum{
		MetricName: aws.String(c.MetricName),
		Value: aws.Float64(c.MetricValue),
		Unit: aws.String(c.MetricUnit),
	}
	metricInput := &cloudwatch.PutMetricDataInput{
		Namespace: aws.String(c.Namespace),
		MetricData: []*cloudwatch.MetricDatum{metricData},
	}

	c.Ui.Output(fmt.Sprintf("Writing metric data to CloudWatch:\n%s", metricInput.String()))
	_, err := svc.PutMetricData(metricInput)
	return err
}

func validateReportArgs(c ReportCommand) error {
	if c.AwsRegion == "" {
		return errors.New("ERROR: The argument '--region' is required.")
	}

	if c.Namespace == "" {
		return errors.New("ERROR: The argument '--namespace' is required.")
	}

	if c.MetricName == "" {
		return errors.New("ERROR: The argument '--name' is required.")
	}

	return nil
}

