package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func main() {
	leerEntrada()
}

//cuando analice texto de entrada se iran guardando aca
var listaComandos []string

func leerEntrada() {

	var enviar bool = false
	var concatenar string = ""
	for true {
		lectura := bufio.NewReader(os.Stdin)
		entrada, _ := lectura.ReadString('\n')         // Leer hasta el separador de salto de línea
		eleccion := strings.TrimRight(entrada, "\r\n") // Remover el salto de línea de la entrada del usuario

		if strings.HasSuffix(eleccion, "\\*") { //si al final termina con \*
			concatenar = concatenar + eleccion
			enviar = false
		} else {
			concatenar = concatenar + eleccion
			enviar = true
		}

		if eleccion == "exit" { //para salir
			break
		}

		if enviar == true { //para empezar analizar
			analizador(concatenar)
			imprimirListaComandos() //************************************************************
			vaciarListaComandos()
			concatenar = ""
			enviar = false
		}
	}

}

func analizador(cadena string) {
	var estado int = 0
	var examinarAsci int = 0
	var caracter string = ""

	for i := 0; i < len(cadena); i++ {
		examinar := cadena[i]        //caracter actual de la cadena
		examinarAsci = int(examinar) //numero asci del caracter actual

		switch estado {
		case 0:
			if examinarAsci == 10 { //salto de linea
				estado = 0
			} else if examinarAsci == 32 { //espacio en blanco
				estado = 0
			} else if examinarAsci == 9 { //tabulacion
				estado = 0
			} else if (examinarAsci > 64 && examinarAsci < 91) || (examinarAsci > 96 && examinarAsci < 123) { //letras
				caracter = caracter + string(examinar)
				estado = 1
			} else if examinarAsci > 47 && examinarAsci < 58 { // numeros
				caracter = caracter + string(examinar)
				estado = 1
			} else if examinarAsci == 47 { // /
				caracter = caracter + string(examinar)
				estado = 1
			} else if examinarAsci == 34 { // ""
				caracter = caracter + string(examinar)
				estado = 4 //entra en estado de rutas
			} else if examinarAsci == 46 { // .
				caracter = caracter + string(examinar)
				estado = 1
			} else if examinarAsci == 58 { // :
				caracter = caracter + string(examinar)
				estado = 1
			} else if examinarAsci == 45 { // -
				caracter = caracter + string(examinar)
				estado = 2
			} else if examinarAsci == 62 { // >
				caracter = caracter + string(examinar)
				estado = 2
			}
			break
		case 1:
			if (examinarAsci > 64 && examinarAsci < 91) || (examinarAsci > 96 && examinarAsci < 123) { //letras
				caracter = caracter + string(examinar)
				estado = 1
			} else if examinarAsci > 47 && examinarAsci < 58 { // numeros
				caracter = caracter + string(examinar)
				estado = 1
			} else if examinarAsci == 47 { // /
				caracter = caracter + string(examinar)
				estado = 1
			} else if examinarAsci == 34 { // "
				caracter = caracter + string(examinar)
				estado = 1
			} else if examinarAsci == 46 { // .
				caracter = caracter + string(examinar)
				estado = 1
			} else if examinarAsci == 58 { // :
				caracter = caracter + string(examinar)
				estado = 1
			} else {
				analizador2(caracter)
				caracter = ""
				i = i - 1
				estado = 0
			}
			break
		case 2:
			if examinarAsci == 45 { // -
				caracter = caracter + string(examinar)
				estado = 2
			} else if examinarAsci == 62 { // >
				caracter = caracter + string(examinar)
				estado = 2
			} else {
				analizador2(caracter)
				caracter = ""
				i = i - 1
				estado = 0
			}
			break
		case 4:
			if (examinarAsci > 64 && examinarAsci < 91) || (examinarAsci > 96 && examinarAsci < 123) { //letras
				caracter = caracter + string(examinar)
				estado = 4
			} else if examinarAsci > 47 && examinarAsci < 58 { // numeros
				caracter = caracter + string(examinar)
				estado = 4
			} else if examinarAsci == 47 { // /
				caracter = caracter + string(examinar)
				estado = 4
			} else if examinarAsci == 34 { // "
				caracter = caracter + string(examinar) //concateno
				estado = 4
			} else if examinarAsci == 46 { // .
				caracter = caracter + string(examinar)
				estado = 4
			} else if examinarAsci == 32 { //espacio en blanco
				caracter = caracter + " "
				estado = 4
			} else {
				analizador2(caracter)
				caracter = ""
				i = i - 1
				estado = 0
			}
			break

		}
	}

	if caracter != "" {
		analizador2(caracter) //para que envie la ultima palabra
	}

}

func analizador2(cadena string) {
	listaComandos = append(listaComandos, cadena)
}

func imprimirListaComandos() {
	for i := 0; i < len(listaComandos); i++ {
		fmt.Println(listaComandos[i])
	}
}

func vaciarListaComandos() {
	listaComandos = nil
}
