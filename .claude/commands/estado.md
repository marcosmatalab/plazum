---
description: "Informe compacto del estado del repo para revisión externa"
---
Genera un informe compacto y honesto del estado del repositorio, pensado para que alguien que no tiene acceso a la máquina pueda revisarlo. Incluye, en este orden y sin adornos:
1. git log --oneline -8, rama, si hay cambios sin commitear, y si main está sincronizado con origin
2. Salida de go test ./... -count=1 resumida (paquetes ok/fail) y el número total de casos
3. Cobertura del núcleo
4. Recuento: líneas de producto, líneas de test, dependencias externas
5. De ETAPAS.md: casillas marcadas y totales por etapa, y la siguiente sin marcar
6. Salida de go run ./cmd/plazum cobertura paquetes
7. Lo que sabes que está roto, a medias o afirmado sin sostener, aunque nadie te lo haya preguntado. Esta sección es obligatoria y no puede decir "nada".
