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
	Type   string `yaml:"type"` // "grid" pour l'instant, "compiegne" plus tard
	Width  int    `yaml:"width"`
	Height int    `yaml:"height"`
}

type CarsConfig struct {
	Count          int     `yaml:"count"`
	InitialBattery float64 `yaml:"initial_battery"` // entre 0 et 1
	MaxBattery     float64 `yaml:"max_battery"`
	Consumption    float64 `yaml:"consumption"`
}

type StationsConfig struct {
	PumpsPerStation int `yaml:"pumps_per_station"`
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
	if c.Cars.Count < 0 {
		return fmt.Errorf("cars.count doit être >= 0, reçu %d", c.Cars.Count)
	}
	if c.Cars.InitialBattery < 0 || c.Cars.InitialBattery > 1 {
		return fmt.Errorf("cars.initial_battery doit être entre 0 et 1, reçu %f", c.Cars.InitialBattery)
	}
	if c.Cars.MaxBattery <= 0 {
		return fmt.Errorf("cars.max_battery doit être > 0, reçu %f", c.Cars.MaxBattery)
	}
	if c.Cars.Consumption <= 0 {
		return fmt.Errorf("cars.consumption doit être > 0, reçu %f", c.Cars.Consumption)
	}
	if c.Stations.PumpsPerStation <= 0 {
		return fmt.Errorf("stations.pumps_per_station doit être > 0, reçu %d", c.Stations.PumpsPerStation)
	}
	return nil
}
