package e2e

import (
	"context"
	"fmt"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
)

// InfluxClient is a small helper around the official InfluxDB v2 client
// used by the E2E tests. It exposes write and query helpers and hides
// token/org/bucket plumbing.
type InfluxClient struct {
	url    string
	org    string
	bucket string
	token  string
	client influxdb2.Client
	write  api.WriteAPIBlocking
	query  api.QueryAPI
}

// NewInfluxClient creates a new client for the given parameters. It assumes
// the server is already running and reachable.
func NewInfluxClient(url, org, bucket, token string) *InfluxClient {
	c := influxdb2.NewClient(url, token)
	return &InfluxClient{
		url:    url,
		org:    org,
		bucket: bucket,
		token:  token,
		client: c,
		write:  c.WriteAPIBlocking(org, bucket),
		query:  c.QueryAPI(org),
	}
}

// WritePoint writes a simple measurement with provided fields and tags.
func (c *InfluxClient) WritePoint(ctx context.Context, measurement string, tags map[string]string, fields map[string]interface{}, ts time.Time) error {
	p := influxdb2.NewPoint(measurement, tags, fields, ts)
	return c.write.WritePoint(ctx, p)
}

// Query runs a Flux query and returns the raw query.Result iterator. The
// caller is responsible for iterating and closing it.
func (c *InfluxClient) Query(ctx context.Context, flux string) (*api.QueryTableResult, error) {
	return c.query.Query(ctx, flux)
}

// SetupBucket ensures the organisation and bucket exist on the running
// InfluxDB instance. It creates them if missing using the management API.
func (c *InfluxClient) SetupBucket(ctx context.Context) error {
	orgAPI := c.client.OrganizationsAPI()
	org, err := orgAPI.FindOrganizationByName(ctx, c.org)
	if err != nil || org == nil {
		org, err = orgAPI.CreateOrganizationWithName(ctx, c.org)
		if err != nil {
			return fmt.Errorf("create org: %w", err)
		}
	}

	bucketAPI := c.client.BucketsAPI()
	buckets, err := bucketAPI.FindBucketsByOrgName(ctx, c.org)
	if err != nil {
		return err
	}
	if buckets != nil {
		for _, b := range *buckets {
			if b.Name == c.bucket {
				return nil
			}
		}
	}
	_, err = bucketAPI.CreateBucketWithName(ctx, org, c.bucket)
	if err != nil {
		return fmt.Errorf("create bucket: %w", err)
	}
	return nil
}

// Close releases the underlying client resources.
func (c *InfluxClient) Close() { c.client.Close() }
