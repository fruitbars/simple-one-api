package config

import (
	"encoding/json"
	"reflect"
	"strings"
)

// The visual admin intentionally edits only the common configuration surface.
// Extensions retain fields introduced by plugins or future releases so a
// visual edit or an Advanced JSON round trip does not silently delete them.

func (value Limit) MarshalJSON() ([]byte, error) {
	type alias Limit
	return marshalWithExtensions(alias(value), value.Extensions)
}

func (value *Limit) UnmarshalJSON(data []byte) error {
	type alias Limit
	return unmarshalWithExtensions(data, (*alias)(value), &value.Extensions)
}

func (value Range) MarshalJSON() ([]byte, error) {
	type alias Range
	return marshalWithExtensions(alias(value), value.Extensions)
}

func (value *Range) UnmarshalJSON(data []byte) error {
	type alias Range
	return unmarshalWithExtensions(data, (*alias)(value), &value.Extensions)
}

func (value ModelParams) MarshalJSON() ([]byte, error) {
	type alias ModelParams
	return marshalWithExtensions(alias(value), value.Extensions)
}

func (value *ModelParams) UnmarshalJSON(data []byte) error {
	type alias ModelParams
	return unmarshalWithExtensions(data, (*alias)(value), &value.Extensions)
}

func (value ServiceModel) MarshalJSON() ([]byte, error) {
	type alias ServiceModel
	return marshalWithExtensions(alias(value), value.Extensions)
}

func (value *ServiceModel) UnmarshalJSON(data []byte) error {
	type alias ServiceModel
	return unmarshalWithExtensions(data, (*alias)(value), &value.Extensions)
}

// ModelDetails embeds ServiceModel, whose JSON methods would otherwise be
// promoted and hide the routing metadata owned by ModelDetails.
func (value ModelDetails) MarshalJSON() ([]byte, error) {
	payload, err := json.Marshal(value.ServiceModel)
	if err != nil {
		return nil, err
	}
	var document map[string]json.RawMessage
	if err := json.Unmarshal(payload, &document); err != nil {
		return nil, err
	}
	document["service_name"], _ = json.Marshal(value.ServiceName)
	document["service_id"], _ = json.Marshal(value.ServiceID)
	return json.Marshal(document)
}

func (value *ModelDetails) UnmarshalJSON(data []byte) error {
	var service ServiceModel
	if err := json.Unmarshal(data, &service); err != nil {
		return err
	}
	var routing struct {
		ServiceName string `json:"service_name"`
		ServiceID   string `json:"service_id"`
	}
	if err := json.Unmarshal(data, &routing); err != nil {
		return err
	}
	delete(service.Extensions, "service_name")
	delete(service.Extensions, "service_id")
	if len(service.Extensions) == 0 {
		service.Extensions = nil
	}
	value.ServiceModel = service
	value.ServiceName = routing.ServiceName
	value.ServiceID = routing.ServiceID
	return nil
}

func (value ProxyConf) MarshalJSON() ([]byte, error) {
	type alias ProxyConf
	return marshalWithExtensions(alias(value), value.Extensions)
}

func (value *ProxyConf) UnmarshalJSON(data []byte) error {
	type alias ProxyConf
	return unmarshalWithExtensions(data, (*alias)(value), &value.Extensions)
}

func (value Translation) MarshalJSON() ([]byte, error) {
	type alias Translation
	return marshalWithExtensions(alias(value), value.Extensions)
}

func (value *Translation) UnmarshalJSON(data []byte) error {
	type alias Translation
	return unmarshalWithExtensions(data, (*alias)(value), &value.Extensions)
}

func (value APIKeyConfig) MarshalJSON() ([]byte, error) {
	type alias APIKeyConfig
	return marshalWithExtensions(alias(value), value.Extensions)
}

func (value *APIKeyConfig) UnmarshalJSON(data []byte) error {
	type alias APIKeyConfig
	return unmarshalWithExtensions(data, (*alias)(value), &value.Extensions)
}

func (value Configuration) MarshalJSON() ([]byte, error) {
	type alias Configuration
	return marshalWithExtensions(alias(value), value.Extensions)
}

func (value *Configuration) UnmarshalJSON(data []byte) error {
	type alias Configuration
	return unmarshalWithExtensions(data, (*alias)(value), &value.Extensions)
}

func marshalWithExtensions(known interface{}, extensions map[string]interface{}) ([]byte, error) {
	payload, err := json.Marshal(known)
	if err != nil {
		return nil, err
	}
	if len(extensions) == 0 {
		return payload, nil
	}
	var document map[string]json.RawMessage
	if err := json.Unmarshal(payload, &document); err != nil {
		return nil, err
	}
	knownFields := jsonFieldNames(reflect.TypeOf(known))
	for key, value := range extensions {
		if _, reserved := knownFields[key]; reserved {
			continue
		}
		raw, err := json.Marshal(value)
		if err != nil {
			return nil, err
		}
		document[key] = raw
	}
	return json.Marshal(document)
}

func unmarshalWithExtensions(data []byte, target interface{}, extensions *map[string]interface{}) error {
	if err := json.Unmarshal(data, target); err != nil {
		return err
	}
	var document map[string]json.RawMessage
	if err := json.Unmarshal(data, &document); err != nil {
		return err
	}
	for field := range jsonFieldNames(reflect.TypeOf(target)) {
		delete(document, field)
	}
	if len(document) == 0 {
		*extensions = nil
		return nil
	}
	extra := make(map[string]interface{}, len(document))
	for key, raw := range document {
		var value interface{}
		if err := json.Unmarshal(raw, &value); err != nil {
			return err
		}
		extra[key] = value
	}
	*extensions = extra
	return nil
}

func jsonFieldNames(valueType reflect.Type) map[string]struct{} {
	for valueType.Kind() == reflect.Pointer {
		valueType = valueType.Elem()
	}
	fields := make(map[string]struct{}, valueType.NumField())
	for index := 0; index < valueType.NumField(); index++ {
		tag := strings.Split(valueType.Field(index).Tag.Get("json"), ",")[0]
		if tag != "" && tag != "-" {
			fields[tag] = struct{}{}
		}
	}
	return fields
}
