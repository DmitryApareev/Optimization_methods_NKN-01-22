package optimizer

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/Knetic/govaluate"
)

// Func — интерфейс для абстрактной функции f(x)
type Func interface {
	Eval(x float64) (float64, error)
}

// evalFunc — реализация Func на основе govaluate
type evalFunc struct {
	expr   *govaluate.EvaluableExpression
	params map[string]interface{}
}

// NewEvalFunc создаёт вычислимую функцию по строке f(x)
func NewEvalFunc(expr string) (Func, error) {
	funcs := map[string]govaluate.ExpressionFunction{
		"sin": func(args ...interface{}) (interface{}, error) { return math.Sin(toFloat(args[0])), nil },
		"cos": func(args ...interface{}) (interface{}, error) { return math.Cos(toFloat(args[0])), nil },
		"tan": func(args ...interface{}) (interface{}, error) { return math.Tan(toFloat(args[0])), nil },
		"exp": func(args ...interface{}) (interface{}, error) { return math.Exp(toFloat(args[0])), nil },
		"log": func(args ...interface{}) (interface{}, error) { return math.Log(toFloat(args[0])), nil },
		"sqrt": func(args ...interface{}) (interface{}, error) {
			return math.Sqrt(toFloat(args[0])), nil
		},
		"abs": func(args ...interface{}) (interface{}, error) {
			return math.Abs(toFloat(args[0])), nil
		},
		"pow": func(args ...interface{}) (interface{}, error) {
			return math.Pow(toFloat(args[0]), toFloat(args[1])), nil
		},
	}

	// нормализуем запятые в десятичной записи
	expr = strings.ReplaceAll(expr, ",", ".")

	parsed, err := govaluate.NewEvaluableExpressionWithFunctions(expr, funcs)
	if err != nil {
		return nil, err
	}

	return &evalFunc{
		expr:   parsed,
		params: map[string]interface{}{"x": 0.0},
	}, nil
}

func (f *evalFunc) Eval(x float64) (float64, error) {
	f.params["x"] = x
	v, err := f.expr.Evaluate(f.params)
	if err != nil {
		return math.NaN(), err
	}

	switch t := v.(type) {
	case float64:
		return t, nil
	case int:
		return float64(t), nil
	case int64:
		return float64(t), nil
	case string:
		parsed, err := strconv.ParseFloat(t, 64)
		if err != nil {
			return math.NaN(), err
		}
		return parsed, nil
	default:
		return math.NaN(), fmt.Errorf("выражение не вернуло число: %T", v)
	}
}

func toFloat(v interface{}) float64 {
	switch t := v.(type) {
	case float64:
		return t
	case int:
		return float64(t)
	case int64:
		return float64(t)
	case string:
		f, _ := strconv.ParseFloat(t, 64)
		return f
	default:
		return math.NaN()
	}
}
