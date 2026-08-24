---
description: "Autoría de corpus: convierte un artículo normativo en obligación con caso dorado, respetando la frontera legal"
---
Flujo de autoría de docs/guia.md §5 y Anexo B (el formato completo, con el ejemplo del art. 31), para el artículo que te pase el usuario. Si la extensión del formato (clase e2e, temporalidad, escalado, pruebas/) aún no está implementada en nucleo/corpus, esa es la PRIMERA casilla de la etapa 3: impleméntala antes según el Anexo B.
1. Pregunta (si no lo dice) la norma y el estrato del paquete (transcrito/referencial/importado/delegado).
2. FRONTERA LEGAL primero: BOE/DOUE se transcribe entero con fuente enlazada; ISO/PCI/SOC 2/TISAX solo identificador y título corto ≤120 caracteres (el linter lo impone); CIS/STIG no se distribuye (delegado). Con ISO está PROHIBIDO procesarla con el modelo: el referencial se escribe del índice público a mano.
3. Aísla las obligaciones (una por verbo exigible), escribe el JSON con id, clase primaria y facetas, cita exacta, vigencia, temporalidad con régimen, escalado, entregable y preguntas.
4. Escribe MÍNIMO 3 casos dorados por reloj (normal, borde de calendario, modificado), derivados DEL TEXTO con su cita_del_esperado, nunca de la implementación.
5. Pasa el linter (go test . -run TestTodosLosPaquetes) y muestra la cobertura con: go run ./cmd/dutiq cobertura paquetes
