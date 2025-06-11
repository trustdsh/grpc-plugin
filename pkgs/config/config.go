package config

import (
	"encoding/json"
	"log/slog"
	"math"

	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

type LoggerOptions struct {
	Type       string
	Level      *slog.Level
	Attributes []slog.Attr
}

func (l *LoggerOptions) MarshalJSON() ([]byte, error) {
	data := map[string]interface{}{
		"type": l.Type,
	}
	if l.Level != nil {
		data["level"] = int(*l.Level)
	}
	attributes := map[string]interface{}{}
	for _, attr := range l.Attributes {
		attributes[attr.Key] = attr.Value.Any()
	}
	data["attributes"] = attributes
	return json.Marshal(data)
}

func (l *LoggerOptions) UnmarshalJSON(data []byte) error {
	var tmp map[string]interface{}
	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}
	l.Type = tmp["type"].(string)
	levelRaw, ok := tmp["level"]
	if ok {
		levelFloat, ok := levelRaw.(float64)
		if !ok {
			return errors.Errorf("level is not an integer: %v", levelRaw)
		}
		levelInt := int(math.Round(levelFloat))
		level := slog.Level(levelInt)
		l.Level = &level
	}
	attrRaw, ok := tmp["attributes"]
	if ok {
		attrMap, ok := attrRaw.(map[string]interface{})
		if !ok {
			return errors.Errorf("attributes is not a map: %v", attrRaw)
		}
		l.Attributes = make([]slog.Attr, 0, len(attrMap))
		for key, value := range attrMap {
			l.Attributes = append(l.Attributes, slog.Attr{Key: key, Value: slog.AnyValue(value)})
		}
	}
	return nil
}

type Config[T any] struct {
	Manifest        *Manifest
	LoggerOptions   *LoggerOptions
	PluginGenerator func(conn grpc.ClientConnInterface) T
}
