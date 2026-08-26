module plazum

go 1.24

require github.com/digitorus/timestamp v0.0.0-20250524132541-c45532741eea

// pkcs7 ya no se importa desde ningun fichero nuestro: vive vendorizado en
// adaptadores/tsa/internal/pkcs7 (ver su LEEME.md, con la procedencia y el
// commit exacto).
//
// ESTA LINEA NO SE PUEDE BORRAR aunque diga "indirect", y el motivo CAMBIO el
// 26-08-2026, asi que conviene leer el nuevo antes de decidir nada.
//
// El motivo VIEJO era grave: timestamp.Parse llamaba al pkcs7 de aguas arriba
// sobre los mismos bytes del expediente, asi que la version de 2023 (la que
// entra en panico con dos bytes, 0x30 0x84, en ber2der) era ALCANZABLE por
// cualquiera que mandara un token roto. Eso se acabo: el TSTInfo se lee ahora
// con encoding/asn1 en adaptadores/tsa/rfc3161.go, sobre el contenido de la
// copia vendorizada, y ningun codigo nuestro llega ya al pkcs7 transitivo.
//
// El motivo QUE QUEDA es mas pequeno y sigue bastando: timestamp sigue
// importando pkcs7, asi que el paquete ENTRA EN EL BINARIO aunque no se llame.
// Distribuir una version con un fallo conocido, aunque hoy sea inalcanzable, es
// lo que un analisis de composicion de software le senala al comprador, y "no
// lo llamamos" no es algo que el comprador pueda comprobar sin leerse el
// codigo.
//
// El arreglo de verdad es el OBJETIVO DECLARADO en DEPENDENCIAS.md: construir
// el TimeStampReq nosotros (unas treinta lineas de ASN.1), quitar timestamp, y
// con ella pkcs7 del grafo.
//
// Lo vigila TestElPkcs7TransitivoNoEsElQueRevienta, en adaptadores/tsa, que
// mira el comportamiento y no el numero de version.
require github.com/digitorus/pkcs7 v0.0.0-20250729175123-57bd227bfa2f // indirect
