package weather

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// ─── Types ────────────────────────────────────────────────────────────────────

type location struct {
	Latitude  float64
	Longitude float64
	City      string
}

type weatherData struct {
	Current struct {
		Temperature2m       float64 `json:"temperature_2m"`
		ApparentTemperature float64 `json:"apparent_temperature"`
		WeatherCode         int     `json:"weather_code"`
		RelativeHumidity2m  int     `json:"relative_humidity_2m"`
		SurfacePressure     float64 `json:"surface_pressure"`
		WindSpeed10m        float64 `json:"wind_speed_10m"`
		WindDirection10m    float64 `json:"wind_direction_10m"`
		WindGusts10m        float64 `json:"wind_gusts_10m"`
		Rain                float64 `json:"rain"`
		Snowfall            float64 `json:"snowfall"`
		UVIndex             float64 `json:"uv_index"`
	} `json:"current"`
	Hourly struct {
		Time                     []string  `json:"time"`
		Temperature2m            []float64 `json:"temperature_2m"`
		PrecipitationProbability []int     `json:"precipitation_probability"`
		WeatherCode              []int     `json:"weather_code"`
	} `json:"hourly"`
	Daily struct {
		Time                        []string  `json:"time"`
		Temperature2mMax            []float64 `json:"temperature_2m_max"`
		Temperature2mMin            []float64 `json:"temperature_2m_min"`
		WeatherCode                 []int     `json:"weather_code"`
		PrecipitationProbabilityMax []int     `json:"precipitation_probability_max"`
		UVIndexMax                  []float64 `json:"uv_index_max"`
	} `json:"daily"`
}

type airData struct {
	Current struct {
		USAQI float64 `json:"us_aqi"`
		PM25  float64 `json:"pm2_5"`
		PM10  float64 `json:"pm10"`
	} `json:"current"`
}

// ─── Styles ───────────────────────────────────────────────────────────────────

var (
	bold    = lipgloss.NewStyle().Bold(true)
	muted   = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	green   = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	red     = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	yellow  = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	cyan    = lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
	magenta = lipgloss.NewStyle().Foreground(lipgloss.Color("5"))
)

// ─── Location ─────────────────────────────────────────────────────────────────

func geolocateIP() (location, error) {
	// Try ip-api.com first (generous free tier, no key required).
	if loc, err := geolocateIPAPI(); err == nil && loc.City != "" {
		return loc, nil
	}
	// Fallback to ipapi.co.
	return geolocateIPAPICo()
}

func geolocateIPAPI() (location, error) {
	resp, err := http.Get("http://ip-api.com/json/?fields=status,city,regionName,lat,lon")
	if err != nil {
		return location{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return location{}, fmt.Errorf("ip-api.com: status %d", resp.StatusCode)
	}
	var data struct {
		Status     string  `json:"status"`
		City       string  `json:"city"`
		RegionName string  `json:"regionName"`
		Lat        float64 `json:"lat"`
		Lon        float64 `json:"lon"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return location{}, err
	}
	if data.Status != "success" || data.City == "" {
		return location{}, fmt.Errorf("ip-api.com: %s", data.Status)
	}
	return location{
		Latitude:  data.Lat,
		Longitude: data.Lon,
		City:      data.City + ", " + data.RegionName,
	}, nil
}

func geolocateIPAPICo() (location, error) {
	resp, err := http.Get("https://ipapi.co/json/")
	if err != nil {
		return location{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return location{}, fmt.Errorf("ipapi.co: status %d", resp.StatusCode)
	}
	var data struct {
		Latitude   float64 `json:"latitude"`
		Longitude  float64 `json:"longitude"`
		City       string  `json:"city"`
		RegionCode string  `json:"region_code"`
		Error      bool    `json:"error"`
		Reason     string  `json:"reason"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return location{}, err
	}
	if data.Error {
		return location{}, fmt.Errorf("ipapi.co: %s", data.Reason)
	}
	if data.City == "" || (data.Latitude == 0 && data.Longitude == 0) {
		return location{}, fmt.Errorf("ipapi.co: empty location response")
	}
	return location{
		Latitude:  data.Latitude,
		Longitude: data.Longitude,
		City:      data.City + ", " + data.RegionCode,
	}, nil
}

func geocodeCity(name string) (location, error) {
	u := "https://geocoding-api.open-meteo.com/v1/search?name=" + url.QueryEscape(name) + "&count=1&language=en&format=json"
	resp, err := http.Get(u)
	if err != nil {
		return location{}, err
	}
	defer resp.Body.Close()
	var data struct {
		Results []struct {
			Latitude  float64 `json:"latitude"`
			Longitude float64 `json:"longitude"`
			Name      string  `json:"name"`
			Admin1    string  `json:"admin1"`
			Country   string  `json:"country"`
		} `json:"results"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return location{}, err
	}
	if len(data.Results) == 0 {
		return location{}, fmt.Errorf("city not found: %s", name)
	}
	r := data.Results[0]
	region := r.Admin1
	if region == "" {
		region = r.Country
	}
	return location{
		Latitude:  r.Latitude,
		Longitude: r.Longitude,
		City:      r.Name + ", " + region,
	}, nil
}

// ─── API ──────────────────────────────────────────────────────────────────────

func fetchWeather(lat, lon float64) (*weatherData, error) {
	params := url.Values{
		"latitude":         {fmt.Sprintf("%f", lat)},
		"longitude":        {fmt.Sprintf("%f", lon)},
		"current":          {"temperature_2m,apparent_temperature,weather_code,relative_humidity_2m,surface_pressure,wind_speed_10m,wind_direction_10m,wind_gusts_10m,rain,snowfall,uv_index"},
		"hourly":           {"temperature_2m,precipitation_probability,weather_code"},
		"daily":            {"temperature_2m_max,temperature_2m_min,weather_code,precipitation_probability_max,uv_index_max"},
		"temperature_unit": {"fahrenheit"},
		"wind_speed_unit":  {"mph"},
		"forecast_days":    {"5"},
		"timezone":         {"auto"},
	}
	resp, err := http.Get("https://api.open-meteo.com/v1/forecast?" + params.Encode())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var w weatherData
	if err := json.NewDecoder(resp.Body).Decode(&w); err != nil {
		return nil, err
	}
	return &w, nil
}

func fetchAir(lat, lon float64) (*airData, error) {
	params := url.Values{
		"latitude":  {fmt.Sprintf("%f", lat)},
		"longitude": {fmt.Sprintf("%f", lon)},
		"current":   {"us_aqi,pm2_5,pm10"},
	}
	resp, err := http.Get("https://air-quality-api.open-meteo.com/v1/air-quality?" + params.Encode())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var a airData
	if err := json.NewDecoder(resp.Body).Decode(&a); err != nil {
		return nil, err
	}
	return &a, nil
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func weatherIcon(code int) string {
	switch {
	case code == 0:
		return "☀️ "
	case code <= 3:
		return "⛅"
	case code <= 49:
		return "☁️ "
	case code <= 69:
		return "🌧️"
	case code <= 79:
		return "❄️ "
	case code <= 82:
		return "🌧️"
	case code <= 99:
		return "⚡"
	default:
		return "❓"
	}
}

func weatherDesc(code int) string {
	switch {
	case code == 0:
		return "Clear sky"
	case code == 1:
		return "Mainly clear"
	case code == 2:
		return "Partly cloudy"
	case code == 3:
		return "Overcast"
	case code <= 49:
		return "Fog"
	case code <= 55:
		return "Drizzle"
	case code <= 57:
		return "Freezing drizzle"
	case code <= 65:
		return "Rain"
	case code <= 67:
		return "Freezing rain"
	case code <= 75:
		return "Snowfall"
	case code == 77:
		return "Snow grains"
	case code <= 82:
		return "Rain showers"
	case code == 85, code == 86:
		return "Snow showers"
	case code == 95:
		return "Thunderstorm"
	case code <= 99:
		return "Thunderstorm with hail"
	default:
		return "Unknown"
	}
}

func windDir(deg float64) string {
	dirs := []string{"N", "NNE", "NE", "ENE", "E", "ESE", "SE", "SSE",
		"S", "SSW", "SW", "WSW", "W", "WNW", "NW", "NNW"}
	return dirs[int(math.Round(deg/22.5))%16]
}

type level struct{ label, color string }

func uvLevel(uv float64) level {
	switch {
	case uv <= 2:
		return level{"Low", "2"}
	case uv <= 5:
		return level{"Moderate", "3"}
	case uv <= 7:
		return level{"High", "1"}
	case uv <= 10:
		return level{"Very High", "5"}
	default:
		return level{"Extreme", "1"}
	}
}

func aqiLevel(aqi float64) level {
	switch {
	case aqi <= 50:
		return level{"Good", "2"}
	case aqi <= 100:
		return level{"Moderate", "3"}
	case aqi <= 150:
		return level{"Unhealthy (Sensitive)", "1"}
	case aqi <= 200:
		return level{"Unhealthy", "1"}
	default:
		return level{"Hazardous", "5"}
	}
}

func colorLevel(l level) string {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(l.color)).Render(l.label)
}

// ─── Render ───────────────────────────────────────────────────────────────────

func render(loc location, w *weatherData, air *airData) string {
	var sb strings.Builder

	sb.WriteString("\n")
	sb.WriteString(bold.Render("📍 "+loc.City) + " " +
		muted.Render(fmt.Sprintf("(%.2f, %.2f)", loc.Latitude, loc.Longitude)) + "\n\n")

	// Current
	cur := w.Current
	uv := uvLevel(cur.UVIndex)
	sb.WriteString(bold.Render(weatherIcon(cur.WeatherCode)+"  Current Conditions") + "\n")
	sb.WriteString(fmt.Sprintf("  %s (feels like %d°F) — %s\n",
		bold.Render(fmt.Sprintf("%d°F", int(math.Round(cur.Temperature2m)))),
		int(math.Round(cur.ApparentTemperature)),
		weatherDesc(cur.WeatherCode),
	))
	sb.WriteString(fmt.Sprintf("  Humidity: %d%%  |  Pressure: %.0f hPa  |  UV: %d (%s)\n\n",
		cur.RelativeHumidity2m, cur.SurfacePressure,
		int(math.Round(cur.UVIndex)), colorLevel(uv),
	))

	// Wind
	sb.WriteString(bold.Render("💨 Wind") + "\n")
	sb.WriteString(fmt.Sprintf("  %d mph %s, gusts to %d mph\n\n",
		int(math.Round(cur.WindSpeed10m)),
		windDir(cur.WindDirection10m),
		int(math.Round(cur.WindGusts10m)),
	))

	// Precipitation
	sb.WriteString(bold.Render("🌧️  Precipitation") + "\n")
	if cur.Rain > 0 {
		sb.WriteString(fmt.Sprintf("  Rain: %.1f mm", cur.Rain))
	} else {
		sb.WriteString("  No rain")
	}
	if cur.Snowfall > 0 {
		sb.WriteString(fmt.Sprintf("  |  Snow: %.1f cm", cur.Snowfall))
	} else {
		sb.WriteString("  |  No snow")
	}
	sb.WriteString("\n\n")

	// Air quality
	if air != nil {
		aqi := aqiLevel(air.Current.USAQI)
		sb.WriteString(bold.Render("🌿 Air Quality") + "\n")
		sb.WriteString(fmt.Sprintf("  AQI: %.0f (%s)  |  PM2.5: %.1f  |  PM10: %.1f\n\n",
			air.Current.USAQI, colorLevel(aqi),
			air.Current.PM25, air.Current.PM10,
		))
	}

	// Hourly forecast
	now := time.Now()
	startIdx := -1
	for i, t := range w.Hourly.Time {
		parsed, err := time.Parse("2006-01-02T15:04", t)
		if err == nil && !parsed.Before(now) {
			startIdx = i
			break
		}
	}
	if startIdx >= 0 {
		sb.WriteString(bold.Render("⏱️  Next 12 Hours") + "\n")
		end := startIdx + 12
		if end > len(w.Hourly.Time) {
			end = len(w.Hourly.Time)
		}
		for i := startIdx; i < end; i++ {
			t, _ := time.Parse("2006-01-02T15:04", w.Hourly.Time[i])
			rain := w.Hourly.PrecipitationProbability[i]
			rainStr := muted.Render("0% rain")
			if rain > 0 {
				rainStr = cyan.Render(fmt.Sprintf("%d%% rain", rain))
			}
			sb.WriteString(fmt.Sprintf("  %5s  %3d°F  %s   %s\n",
				t.Format("3PM"),
				int(math.Round(w.Hourly.Temperature2m[i])),
				weatherIcon(w.Hourly.WeatherCode[i]),
				rainStr,
			))
		}
		sb.WriteString("\n")
	}

	// Daily forecast
	sb.WriteString(bold.Render("📅 5-Day Forecast") + "\n")
	for i, d := range w.Daily.Time {
		date, _ := time.Parse("2006-01-02", d)
		rain := w.Daily.PrecipitationProbabilityMax[i]
		rainStr := muted.Render("0% rain")
		if rain > 0 {
			rainStr = cyan.Render(fmt.Sprintf("%d%% rain", rain))
		}
		sb.WriteString(fmt.Sprintf("  %s  %d/%d°F  %s   %s  UV:%d\n",
			date.Format("Mon"),
			int(math.Round(w.Daily.Temperature2mMin[i])),
			int(math.Round(w.Daily.Temperature2mMax[i])),
			weatherIcon(w.Daily.WeatherCode[i]),
			rainStr,
			int(math.Round(w.Daily.UVIndexMax[i])),
		))
	}

	// Suppress unused style warnings — these are used via colorLevel() but
	// referenced here to keep them in scope.
	_ = red
	_ = yellow
	_ = green
	_ = magenta

	return sb.String()
}

// ─── Run ──────────────────────────────────────────────────────────────────────

func Run(city string) error {
	var (
		loc location
		err error
	)
	if city != "" {
		loc, err = geocodeCity(city)
	} else {
		loc, err = geolocateIP()
	}
	if err != nil {
		return fmt.Errorf("location: %w", err)
	}

	w, err := fetchWeather(loc.Latitude, loc.Longitude)
	if err != nil {
		return fmt.Errorf("weather: %w", err)
	}

	air, _ := fetchAir(loc.Latitude, loc.Longitude)

	fmt.Print(render(loc, w, air))
	return nil
}
