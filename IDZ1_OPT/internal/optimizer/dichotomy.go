package optimizer

import "errors"

// Iter — одна итерация метода дихотомии
type Iter struct {
	K     int     `json:"k"`
	A     float64 `json:"a"`
	B     float64 `json:"b"`
	XMid  float64 `json:"xmid"`
	FXMid float64 `json:"fxmid"`
	Len   float64 `json:"len"`
}

// ErrStopped — специальная ошибка для принудительной остановки
var ErrStopped = errors.New("dichotomy: stopped by callback")

// Dichotomy — реализация метода дихотомии
// onIter вызывается после каждой итерации; если вернёт ErrStopped — алгоритм прерывается.
func Dichotomy(
	f Func,
	a, b, eps, delta float64,
	maxIter int,
	onIter func(Iter) error,
) (Iter, error) {
	var last Iter

	for k := 1; k <= maxIter && (b-a)/2 > eps; k++ {
		x1 := (a + b - delta) / 2
		x2 := (a + b + delta) / 2

		fx1, err1 := f.Eval(x1)
		fx2, err2 := f.Eval(x2)
		if err1 != nil || err2 != nil {
			if err1 != nil {
				return last, err1
			}
			return last, err2
		}

		if fx1 <= fx2 {
			b = x2
		} else {
			a = x1
		}

		mid := (a + b) / 2
		fmid, _ := f.Eval(mid)

		last = Iter{
			K:     k,
			A:     a,
			B:     b,
			XMid:  mid,
			FXMid: fmid,
			Len:   b - a,
		}

		if onIter != nil {
			if err := onIter(last); err != nil {
				if errors.Is(err, ErrStopped) {
					return last, ErrStopped
				}
				return last, err
			}
		}
	}

	return last, nil
}
