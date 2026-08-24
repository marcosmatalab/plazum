// Package adaptadores contiene las implementaciones de los puertos.
//
// Estado: por construir, etapa a etapa (ver ETAPAS.md). Cada adaptador vive
// en su subdirectorio, declara sus dependencias en DEPENDENCIAS.md y jamas
// importa nada que escriba estado saltandose su puerto.
//
//	sqlite/     Almacen de referencia, blobs cifrados dentro     (etapa 1)
//	tsa/        Anclaje RFC 3161 con cadena de reserva           (etapa 1)
//	notifica/   email y Teams (etapa 4), Slack y Jira (etapa 6);
//	            en la etapa 2 solo el smoke test del canal del latido
//	oidc/       Identidad: OIDC + SCIM con manager               (etapa 2)
//	wasm/       host Extism para conectores                      (etapa 6)
//	delegados/  Prowler, OpenSCAP, Trivy, ScubaGear via OCSF     (etapa 6)
//	llm/        Asistente fuera de proceso                       (etapa 5)
//	mcp/        server de solo lectura, client, skills           (etapas 5 y 6)
package adaptadores
