package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
)

func main() {
	leerEntrada()
}

//cuando analice texto de entrada se iran guardando aca los comandos
var listaComandos []string

//mbr que tendra cada archivo creado
type mbr struct {
	tamanio       uint64
	fecha         [20]byte
	numAsignacion uint64
	particiones   []byte
}

//leera los comandos de entrada (los que escribe el usuario)
func leerEntrada() {

	var enviar bool = false
	var concatenar string = ""
	for true {
		lectura := bufio.NewReader(os.Stdin)
		entrada, _ := lectura.ReadString('\n')         // Leer hasta el separador de salto de línea
		eleccion := strings.TrimRight(entrada, "\r\n") // Remover el salto de línea de la entrada del usuario

		if strings.HasSuffix(eleccion, "\\*") { //si al final termina con \* (seguira concatenando)
			concatenar = concatenar + eleccion
			enviar = false
		} else { //cuando la ultima linea no tengo al final \* entonces ejecutara
			concatenar = concatenar + eleccion
			enviar = true
		}

		if eleccion == "exit" { //para salir de ejecucion
			break
		}

		if enviar == true { //para empezar analizar
			analizador(concatenar)
			//imprimirListaComandos() //************************************************************
			logica()
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
			} else if examinarAsci == 35 { // #
				estado = 3 //pasa a estado de comentario
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
		case 3:
			if examinarAsci == 10 { //salto de linea
				//i = i - 1
				estado = 0
			} else {
				estado = 3
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

//ira guardando comandos en la lista
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

func logica() {
	for i := 0; i < len(listaComandos); i++ {
		if strings.ToLower(listaComandos[i]) == "exec" {
			execComando(i)
		} else if strings.ToLower(listaComandos[i]) == "mkdisk" {
			mkdiskComando(i)
		}
	}
}

//--------------------------------INICIO EXEC-------------------------------//
//recibe el parametro index, que es el indice por donde actualmente se esta
func execComando(index int) {

	for i := index; i < len(listaComandos); i++ {
		if strings.ToLower(listaComandos[i]) == "path" { //cuando encuentre palabra reservada path
			if (strings.Compare(listaComandos[i-1], "-") == 0) && (strings.Compare(listaComandos[i+1], "->") == 0) { // validar si esta de esta forma -path->
				ruta := listaComandos[i+2]        //ruta
				if strings.Contains(ruta, "\"") { //si la ruta que viene contiene comillas
					ruta2 := ruta[1 : len(ruta)-1] //le quitamos comillas a la ruta
					leerArchivoExec(ruta2)         //funcion que leera el archivo
				} else { //sino tiene comillas manda la ruta normal
					leerArchivoExec(ruta)
				}
			} else {
				fmt.Println("\n---> Se ha producido un error con el comando 'exec'")
			}
		}
	}
}

//lee el texto que contiene un archivo
func leerArchivoExec(ruta string) {
	fmt.Println("")
	texto, err := ioutil.ReadFile(ruta) // just pass the file name, ej. /home/gudiel/z.txt
	if err != nil {
		fmt.Print(err)
	}

	//fmt.Println(b)   // imprimir contenido en'bytes'
	str := string(texto) // convertir a 'string'
	//fmt.Println(str)
	lecturaLineaLineaDeArchivo(str)
}

//ira ejecutando los comandos de las lineas leidas
func lecturaLineaLineaDeArchivo(texto string) {
	var estado int = 0
	var examinarAsci int = 0
	var caracteres string = "" //ira concantenando carecteres de linea actual
	//var comandos string = ""   // sera la linea o lineas de comandos

	for i := 0; i < len(texto); i++ {
		examinar := texto[i]         //caracter actual de la cadena
		examinarAsci = int(examinar) //numero asci del caracter actual

		switch estado {
		case 0:
			if examinarAsci == 10 { //salto de linea, quiere decir que finalizo una linea de comandos
				caracteres = caracteres + string(examinar)

				if strings.Contains(strings.ToLower(caracteres), "pause") {
					//fmt.Println(caracteres)
					vaciarListaComandos() //al principio porque el usuario al inicio ejecuta un exec, entonces se vacia para que no entre en ciclo infinito
					analizador(caracteres)
					imprimirListaComandos()
					logica()
					vaciarListaComandos()
					caracteres = ""
					fmt.Println("PAUSE: PRECIOSE PARA CONTINUAR....")
					bufio.NewReader(os.Stdin).ReadBytes('\n')
				}

			} else { //ira concatenando cualquier cosa que no sea salto de linea
				caracteres = caracteres + string(examinar)
			}
		}
	}

	//para que al final envie los ultimos comandos, (ya que de ultimo no abra un proximo pause)
	vaciarListaComandos() //al principio porque el usuario al inicio ejecuta un exec, entonces se vacia para que no entre en ciclo infinito
	analizador(caracteres)
	imprimirListaComandos()
	logica()
	vaciarListaComandos()
	caracteres = ""
}

//--------------------------------FIN EXEC----------------------------//

//-------------------------------INICIO MKDISK-------------------------------//
//MKDISK
func mkdiskComando(index int) {

	var size uint64 = 0
	path := ""
	name := ""
	var unit byte = 'm'
	/*t := time.Now()
	fecha := fmt.Sprintf("%d-%02d-%02dT%02d:%02d:%02d",
		t.Year(), t.Month(), t.Day(),
		t.Hour(), t.Minute(), t.Second())*/

	for i := index; i < len(listaComandos); i++ {

		if strings.ToLower(listaComandos[i]) == "size" {
			if (strings.Compare(listaComandos[i-1], "-") == 0) && (strings.Compare(listaComandos[i+1], "->") == 0) { // validar si esta de esta forma -path->
				tam, err := strconv.Atoi(listaComandos[i+2]) //convierto el valor a int
				size = uint64(tam)
				if err != nil {
					fmt.Print("\nDebe ingresar un numero en size de MKDISK")
				}

			} else {
				fmt.Println("\n---> Se ha producido un error con el comando 'MKDISK' -> 'Size'")
			}

		} else if strings.ToLower(listaComandos[i]) == "path" { //cuando encuentre palabra reservada path
			if (strings.Compare(listaComandos[i-1], "-") == 0) && (strings.Compare(listaComandos[i+1], "->") == 0) { // validar si esta de esta forma -path->
				ruta := listaComandos[i+2]        //ruta
				if strings.Contains(ruta, "\"") { //si la ruta que viene contiene comillas
					ruta2 := ruta[1 : len(ruta)-1] //le quitamos comillas a la ruta
					path = ruta2                   //funcion que leera el archivo
				} else { //sino tiene comillas manda la ruta normal
					path = ruta
				}
			} else {
				fmt.Println("\n---> Se ha producido un error con el comando 'MKDISK' -> 'path'")
			}
		} else if strings.ToLower(listaComandos[i]) == "name" {
			if (strings.Compare(listaComandos[i-1], "-") == 0) && (strings.Compare(listaComandos[i+1], "->") == 0) { // validar si esta de esta forma -path->
				name = listaComandos[i+2] //name
			} else {
				fmt.Println("\n---> Se ha producido un error con el comando 'MKDISK' -> 'name'")
			}
		} else if strings.ToLower(listaComandos[i]) == "unit" {
			if (strings.Compare(listaComandos[i-1], "-") == 0) && (strings.Compare(listaComandos[i+1], "->") == 0) { // validar si esta de esta forma -path->
				cad := listaComandos[i+2] //toma el string
				unit = cad[0]             //extraigo el caracter
			} else {
				fmt.Println("\n---> Se ha producido un error con el comando 'MKDISK' -> 'name'")
			}
		}
	}

	crearArchivo(uint64(size), path, name, unit)

}

func crearArchivo(size uint64, path string, name string, unit byte) {

}

//-------------------------------FIN MKDISK-------------------------------//
