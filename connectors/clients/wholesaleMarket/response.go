package wholesalemarket

import (
	"bytes"
	"fmt"
	"time"

	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
)

type Response struct {
	FrancePowerExchanges []struct {
		StartDate   string `json:"start_date"`
		EndDate     string `json:"end_date"`
		UpdatedDate string `json:"updated_date"`
		Values      []struct {
			StartDate string  `json:"start_date"`
			EndDate   string  `json:"end_date"`
			Value     float64 `json:"value"`
			Price     float64 `json:"price"`
		} `json:"values"`
	} `json:"france_power_exchanges"`
}

func (r *Response) PriceChartHTML() (string, error) {
	line := charts.NewLine()

	// Set chart title and labels
	line.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{Title: "Price Chart"}),
		charts.WithXAxisOpts(opts.XAxis{Name: "Date & Time"}),
		charts.WithYAxisOpts(opts.YAxis{Name: "Price (€/MWh)"}),
	)

	// Prepare data for the chart
	var xAxis []string
	var yAxis []opts.LineData
	for _, exchange := range r.FrancePowerExchanges {
		for _, v := range exchange.Values {
			parsedTime, err := time.Parse(time.RFC3339, v.StartDate)
			if err != nil {
				return "", fmt.Errorf("failed to parse time: %v", err)
			}
			// Include date and time in the X-axis
			xAxis = append(xAxis, parsedTime.Format("2006-01-02 15:04"))
			yAxis = append(yAxis, opts.LineData{Value: v.Price})
		}
	}

	// Add data to the line chart
	line.SetXAxis(xAxis).AddSeries("Price", yAxis)

	// Render the chart to a buffer
	var buf bytes.Buffer
	if err := line.Render(&buf); err != nil {
		return "", fmt.Errorf("failed to render chart: %v", err)
	}

	// Return the HTML as a string
	return buf.String(), nil
}
