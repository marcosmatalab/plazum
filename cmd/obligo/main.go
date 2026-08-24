// Comando obligo: superficie minima del producto.
//
//	obligo verify <expediente.json>   recalcula el expediente entero, sin red
//	obligo explain <expediente.json>  imprime la derivacion de cada conclusion
//	obligo estado  <expediente.json>  los cinco denominadores, nunca un porcentaje
//	obligo cobertura <dir_paquetes>   la cobertura honesta de cada paquete instalado
package main

import (
	"fmt"
	"os"

	"obligo/nucleo/aplicabilidad"
	"obligo/nucleo/corpus"
	"obligo/nucleo/expediente"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "uso: obligo <verify|explain|estado> <expediente.json>")
		fmt.Fprintln(os.Stderr, "     obligo cobertura <dir_paquetes>")
		os.Exit(2)
	}
	if os.Args[1] == "cobertura" {
		ps, err := corpus.Cargar(os.Args[2])
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
		for _, p := range ps {
			fmt.Println(corpus.Medir(p))
		}
		return
	}
	b, err := os.ReadFile(os.Args[2]) // #nosec G304 -- ruta proporcionada por el operador en su maquina
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	e, err := expediente.Cargar(b)
	if err != nil {
		fmt.Fprintln(os.Stderr, "expediente ilegible:", err)
		os.Exit(1)
	}

	switch os.Args[1] {
	case "verify":
		inf := expediente.Verificar(e)
		fmt.Printf("expediente de %s, como estaba el %s\n\n", e.Organizacion, e.ComoEstaba.Format("2006-01-02 15:04 -07:00"))
		for _, c := range inf.Comprobaciones {
			fmt.Println("  ok        ", c)
		}
		for _, d := range inf.Discrepancias {
			fmt.Printf("  DISCREPA   %s\n             declarado:   %s\n             recalculado: %s\n", d.Que, d.Esperado, d.Obtenido)
		}
		fmt.Println()
		if inf.Valido {
			fmt.Println("VERIFICADO. Recalculado desde cero sin red y sin confiar en el emisor.")
			return
		}
		fmt.Printf("NO VERIFICA: %d discrepancia(s).\n", len(inf.Discrepancias))
		os.Exit(1)

	case "explain":
		fmt.Printf("paquetes instalados (%d)\n", len(e.Paquetes))
		for _, p := range e.Paquetes {
			fmt.Printf("  %-20s %-12s vigente desde %s  %s\n", p.URN, p.Clase, p.Vigencia.Desde.Format("2006-01-02"), p.Digest)
		}
		m := aplicabilidad.NuevoMotor()
		for _, pr := range e.Programas {
			m.Cargar(pr)
		}
		for _, h := range e.Hechos {
			m.Afirmar(h)
		}
		if _, err := m.Evaluar(); err != nil {
			fmt.Fprintln(os.Stderr, "aplicabilidad:", err)
			os.Exit(1)
		}
		fmt.Printf("\naplicabilidad derivada\n")
		for _, h := range m.Consultar(aplicabilidad.A("aplica", aplicabilidad.V("O"), aplicabilidad.V("S"))) {
			fmt.Printf("  %-28s sobre %-20s <- %s\n", h.Args[0], h.Args[1], m.Explicar(h))
		}
		fmt.Printf("\nrelojes\n")
		for _, r := range e.Relojes {
			for _, rec := range e.Reclamaciones {
				if rec.Obligacion != r.Obligacion {
					continue
				}
				if rec.Estado == "determinado" {
					fmt.Printf("  %-28s %-18s vence %s\n", r.Obligacion, rec.Hito, rec.Vence.Format("2006-01-02 15:04 -07:00"))
				} else {
					fmt.Printf("  %-28s %-18s %s\n", r.Obligacion, rec.Hito, rec.Estado)
				}
			}
		}

	case "estado":
		fmt.Printf("estado de %s\n\n", e.Organizacion)
		for _, s := range e.Estados {
			fmt.Printf("  %-22s %s\n", s.Prueba, s.Estado)
		}
		d := e.Denominadores
		fmt.Printf("\ncinco denominadores separados, sin porcentaje unico:\n")
		fmt.Printf("  verificable por maquina   %d\n  declarado por humano      %d\n"+
			"  atestado por externo      %d\n  desconocido               %d\n"+
			"  caducado o contradicho    %d\n",
			d.Maquina, d.Humano, d.Externo, d.Desconocido, d.CaducadoOContradicho)

	default:
		fmt.Fprintln(os.Stderr, "orden desconocida:", os.Args[1])
		os.Exit(2)
	}
}
