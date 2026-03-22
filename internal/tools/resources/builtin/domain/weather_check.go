package builtin

import (
	"context"
	"time"

	"goagent/internal/tools/resources/base"
	"goagent/internal/tools/resources/core"
	"goagent/internal/tools/resources/types"
)

// WeatherCheck provides weather information.
type WeatherCheck struct {
	*base.BaseTool
	provider WeatherProvider
}

// WeatherProvider defines the interface for weather data.
type WeatherProvider interface {
	GetCurrent(ctx context.Context, location string) (*types.WeatherData, error)
	GetForecast(ctx context.Context, location string, days int) ([]*types.WeatherData, error)
}

// NewWeatherCheck creates a new WeatherCheck tool.
func NewWeatherCheck(provider WeatherProvider) *WeatherCheck {
	params := &core.ParameterSchema{
		Type: "object",
		Properties: map[string]*core.Parameter{
			"location": {
				Type:        "string",
				Description: "Location (city name or coordinates)",
			},
			"forecast_days": {
				Type:        "integer",
				Description: "Number of forecast days (1-7)",
				Default:     1,
			},
			"units": {
				Type:        "string",
				Description: "Temperature units (celsius/fahrenheit)",
				Default:     "celsius",
				Enum:        []interface{}{"celsius", "fahrenheit"},
			},
		},
		Required: []string{"location"},
	}

	wc := &WeatherCheck{
		provider: provider,
	}
	wc.BaseTool = base.NewBaseTool("weather_check", "Check weather for a location", params)

	return wc
}

// Execute performs the weather check.
func (t *WeatherCheck) Execute(ctx context.Context, params map[string]interface{}) (core.Result, error) {
	location, ok := params["location"].(string)
	if !ok || location == "" {
		return core.NewErrorResult("location is required"), nil
	}

	forecastDays := getInt(params, "forecast_days", 1)
	if forecastDays < 1 {
		forecastDays = 1
	}
	if forecastDays > 7 {
		forecastDays = 7
	}

	if forecastDays == 1 {
		weather, err := t.provider.GetCurrent(ctx, location)
		if err != nil {
			return core.NewErrorResult(err.Error()), nil
		}

		return core.NewResult(true, map[string]interface{}{
			"location":    weather.Location,
			"temperature": weather.Temperature,
			"condition":   weather.Condition,
			"humidity":    weather.Humidity,
			"wind_speed":  weather.WindSpeed,
			"uv_index":    weather.UVIndex,
		}), nil
	}

	// Get forecast
	forecast, err := t.provider.GetForecast(ctx, location, forecastDays)
	if err != nil {
		return core.NewErrorResult(err.Error()), nil
	}

	forecastData := make([]map[string]interface{}, len(forecast))
	for i, day := range forecast {
		forecastData[i] = map[string]interface{}{
			"date":          day.Timestamp.Format("2006-01-02"),
			"temperature":   day.Temperature,
			"condition":     day.Condition,
			"humidity":      day.Humidity,
			"precipitation": day.Precipitation,
		}
	}

	return core.NewResult(true, map[string]interface{}{
		"location": location,
		"forecast": forecastData,
	}), nil
}

// MockWeatherProvider provides mock weather data for testing.
type MockWeatherProvider struct {
	currentTemp float64
	condition   string
}

// NewMockWeatherProvider creates a MockWeatherProvider.
func NewMockWeatherProvider() *MockWeatherProvider {
	return &MockWeatherProvider{
		currentTemp: 22.0,
		condition:   "sunny",
	}
}

// GetCurrent returns mock current weather.
func (m *MockWeatherProvider) GetCurrent(ctx context.Context, location string) (*types.WeatherData, error) {
	return &types.WeatherData{
		Location:    location,
		Temperature: m.currentTemp,
		Condition:   m.condition,
		Humidity:    65,
		WindSpeed:   12.5,
		UVIndex:     5,
		Timestamp:   time.Now(),
	}, nil
}

// GetForecast returns mock forecast.
func (m *MockWeatherProvider) GetForecast(ctx context.Context, location string, days int) ([]*types.WeatherData, error) {
	result := make([]*types.WeatherData, days)
	conditions := []string{"sunny", "cloudy", "rainy", "partly_cloudy"}

	for i := 0; i < days; i++ {
		result[i] = &types.WeatherData{
			Location:      location,
			Temperature:   m.currentTemp + float64(i*2-2),
			Condition:     conditions[i%len(conditions)],
			Humidity:      65 + i*5,
			Precipitation: float64(i * 10),
			Timestamp:     time.Now().Add(time.Duration(i*24) * time.Hour),
		}
	}

	return result, nil
}