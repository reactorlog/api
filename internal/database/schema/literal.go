package schema

import (
	"errors"
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"
)

func formatLiteral(value any) (string, error) {
	v := reflect.ValueOf(value)
	if !v.IsValid() {
		return "NULL", nil
	}
	switch v.Kind() {
	case reflect.String:
		return formatStringLiteral(v.String())
	case reflect.Bool:
		return strconv.FormatBool(v.Bool()), nil
	default:
		return formatNumericLiteral(v)
	}
}

func formatNumericLiteral(value reflect.Value) (string, error) {
	switch value.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(value.Int(), 10), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return strconv.FormatUint(value.Uint(), 10), nil
	case reflect.Float32, reflect.Float64:
		return formatFloatLiteral(value.Float(), value.Type().Bits())
	default:
		return "", fmt.Errorf("unsupported check value type %s", value.Type())
	}
}

func formatFloatLiteral(value float64, bits int) (string, error) {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return "", errors.New("check values must be finite")
	}
	return strconv.FormatFloat(value, 'g', -1, bits), nil
}

func formatStringLiteral(value string) (string, error) {
	if strings.ContainsRune(value, '\x00') {
		return "", errors.New("check strings cannot contain NUL")
	}
	escaped := strings.ReplaceAll(value, `\`, `\\`)
	escaped = strings.ReplaceAll(escaped, "'", "''")
	return "E'" + escaped + "'", nil
}
