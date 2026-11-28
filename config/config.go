// Package config charge et valide la configuration de la simulation depuis un fichier YAML
package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config est la structure racine de la configuration de simulation.
// Les tags `yaml:"..."` disent à la lib YAML comment mapper les clés
// du fichier sur les champs Go (les noms Go sont en PascalCase par
// convention, les clés YAML en snake_case).
type Config struct {
	Seed     int64          `yaml:"seed"`
	MaxTicks int            `yaml:"max_ticks"`
	Map      MapConfig      `yaml:"map"`
	Cars     CarsConfig     `yaml:"cars"`
	Stations StationsConfig `yaml:"stations"`
}

type MapConfig struct {
	Type             string   `yaml:"type"` // "grid" pour l'instant, "compiegne" plus tard
	Width            int      `yaml:"width"`
	Height           int      `yaml:"height"`
	NodeSpacingM     float64  `yaml:"node_spacing_m"`    // distance entre deux noeuds voisins, en mètres
	StationPositions [][2]int `yaml:"station_positions"` // positions [x, y] des stations sur la grille
}

type CarsConfig struct {
	Count               int     `yaml:"count"`
	InitialBattery      float64 `yaml:"initial_battery"` // entre 0 et 1
	MaxBatteryKWh       float64 `yaml:"max_battery_kwh"`
	ConsumptionKWhPerKm float64 `yaml:"consumption_kwh_per_km"`
	BatteryThreshold    float64 `yaml:"battery_threshold"` // fraction [0,1] sous laquelle on cherche une station
}

type StationsConfig struct {
	PumpsPerStation  int     `yaml:"pumps_per_station"`
	PumpPowerKW      float64 `yaml:"pump_power_kw"`       // puissance nominale de chaque borne en kW
	PriceEurosPerKWh float64 `yaml:"price_euros_per_kwh"` // prix facturé en euros par kWh
}

// Load lit et parse un fichier YAML depuis le chemin donné.
// Valide ensuite la cohérence des valeurs (bornes, non-zéros).
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("lecture config %q: %w", path, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse YAML %q: %w", path, err)
	}

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("config invalide %q: %w", path, err)
	}

	return &cfg, nil
}

// validate vérifie que la config est cohérente avant qu'elle soit passée au reste du système.
func (c *Config) validate() error {
	if c.MaxTicks <= 0 {
		return fmt.Errorf("max_ticks doit être > 0, reçu %d", c.MaxTicks)
	}
	if c.Map.Type != "grid" {
		return fmt.Errorf("map.type doit être 'grid' (seul supporté pour l'instant), reçu %q", c.Map.Type)
	}
	if c.Map.Width <= 0 || c.Map.Height <= 0 {
		return fmt.Errorf("map.width et map.height doivent être > 0, reçu %dx%d", c.Map.Width, c.Map.Height)
	}
	if c.Map.NodeSpacingM <= 0 {
		return fmt.Errorf("map.node_spacing_m doit être > 0, reçu %f", c.Map.NodeSpacingM)
	}
	for i, pos := range c.Map.StationPositions {
		x, y := pos[0], pos[1]
		if x < 0 || x >= c.Map.Width || y < 0 || y >= c.Map.Height {
			return fmt.Errorf("map.station_positions[%d] = [%d, %d] hors de la grille %dx%d",
				i, x, y, c.Map.Width, c.Map.Height)
		}
	}
	if c.Cars.Count < 0 {
		return fmt.Errorf("cars.count doit être >= 0, reçu %d", c.Cars.Count)
	}
	if c.Cars.InitialBattery < 0 || c.Cars.InitialBattery > 1 {
		return fmt.Errorf("cars.initial_battery doit être entre 0 et 1, reçu %f", c.Cars.InitialBattery)
	}
	if c.Cars.MaxBatteryKWh <= 0 {
		return fmt.Errorf("cars.max_battery_kwh doit être > 0, reçu %f", c.Cars.MaxBatteryKWh)
	}
	if c.Cars.ConsumptionKWhPerKm <= 0 {
		return fmt.Errorf("cars.consumption_kwh_per_km doit être > 0, reçu %f", c.Cars.ConsumptionKWhPerKm)
	}
	if c.Cars.BatteryThreshold < 0 || c.Cars.BatteryThreshold > 1 {
		return fmt.Errorf("cars.battery_threshold doit être entre 0 et 1, reçu %f", c.Cars.BatteryThreshold)
	}
	if c.Stations.PumpsPerStation <= 0 {
		return fmt.Errorf("stations.pumps_per_station doit être > 0, reçu %d", c.Stations.PumpsPerStation)
	}
	if c.Stations.PumpPowerKW <= 0 {
		return fmt.Errorf("stations.pump_power_kw doit être > 0, reçu %f", c.Stations.PumpPowerKW)
	}
	if c.Stations.PriceEurosPerKWh <= 0 {
		return fmt.Errorf("stations.price_euros_per_kwh doit être > 0, reçu %f", c.Stations.PriceEurosPerKWh)
	}
	return nil
}
