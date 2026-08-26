module plazum

go 1.24

require github.com/digitorus/timestamp v0.0.0-20250524132541-c45532741eea

// pkcs7 ya no se importa desde ningun fichero nuestro: vive vendorizado en
// adaptadores/tsa/internal/pkcs7 (ver su LEEME.md, con la procedencia y el
// commit exacto).
//
// ESTA LINEA NO SE PUEDE BORRAR aunque diga "indirect", y por eso el porque va
// escrito aqui y no en otro sitio. github.com/digitorus/timestamp SIGUE
// importando pkcs7 y su propio go.mod pide la version de 2023, que es la que
// entra en panico con dos bytes (0x30 0x84) en ber2der. Sin esta linea, la
// seleccion de version minima elige la de timestamp y el panico vuelve por la
// puerta de timestamp.Parse, que es lo primero que toca el token del
// expediente.
//
// Lo vigila TestElPkcs7TransitivoNoEsElQueRevienta, en adaptadores/tsa, que
// mira el comportamiento y no el numero de version.
require github.com/digitorus/pkcs7 v0.0.0-20250729175123-57bd227bfa2f // indirect
