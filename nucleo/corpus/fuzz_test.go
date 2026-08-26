package corpus

import (
	"encoding/json"
	"testing"
)

// FuzzValidar: un paquete.json arbitrario jamas debe tumbar el linter. Los
// paquetes son codigo de terceros: se rechazan, no se ejecutan a ver que pasa.
func FuzzValidar(f *testing.F) {
	f.Add([]byte(`{"urn":"x","version":"1","clase":1,"identificador":{"tipo":"eli-ue","valor":"reg/2016/679/oj"},"obligaciones":[{"id":"a","cita":"c","clase_e2e":"documental"}]}`))
	f.Add([]byte(`{`))
	f.Add([]byte(`{"clase":99,"obligaciones":[{}]}`))
	f.Fuzz(func(t *testing.T, data []byte) {
		var p Paquete
		if err := json.Unmarshal(data, &p); err != nil {
			return
		}
		_ = p.Validar() // no debe entrar en panico jamas
	})
}
