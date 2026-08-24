# Política de seguridad

## Divulgación coordinada

Si encuentras una vulnerabilidad, usa el private vulnerability reporting de GitHub (pestaña Security de este repositorio). Compromiso: acuse en 72 horas, evaluación en 7 días, corrección según severidad con crédito al descubridor si lo desea. No abras issues públicos para vulnerabilidades.

## Alcance

El binario obligo, los paquetes de corpus firmados y la web. Los conectores de terceros tienen su propia responsabilidad, acotada por el sandbox WASM (sin red fuera del allowlist, sin filesystem, secretos solo en el host).

## Ventana de soporte

Las dos últimas versiones menores reciben correcciones de seguridad. Cada release publica SBOM (CycloneDX) y va firmada.

## Nuestro propio cumplimiento

Este proyecto aplica el CRA a sí mismo desde su primer hito comercial: SBOM por release, esta política de divulgación, y ventana de soporte declarada. El producto se instala su propio paquete y publica su expediente.
