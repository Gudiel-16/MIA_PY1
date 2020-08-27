package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

func main() {
	leerEntrada()
	//reporteMBR("/home/gudiel/Hoja1_201404278.dsk")

}

//cuando analice texto de entrada se iran guardando aca los comandos
var listaComandos []string

//--------------------------------ESTRUCTURAS-------------------------------//

//mbr que tendra cada archivo creado
type mbr struct {
	Tamanio       int64
	Fecha         [20]byte
	NumAsignacion int64
	Particiones   [4]NodoParticion
}

//NodoParticion ,
type NodoParticion struct {
	Estado             uint8
	TipoParticion      byte
	TipoAjuste         [2]byte
	Tamanio            int64
	Name               [16]byte
	ParticionesLogicas [5]NodoParticionLogica //sera funcional solo para las extendidas
	Start              int64                  //byte donde inicia la particion
}

//NodoParticionLogica ,
type NodoParticionLogica struct {
	Estado        uint8
	TipoParticion byte
	TipoAjuste    [2]byte
	Tamanio       int64
	Name          [16]byte
	Start         int64 //byte donde inicia la particion
	Next          int64 //byte donde termina la particion
}

//--------------------------------FINAL ESTRUCTURAS-------------------------------//

//leera los comandos de entrada (los que escribe el usuario)
func leerEntrada() {

	var enviar bool = false
	var concatenar string = ""
	for true {
		fmt.Print("\n[ nuevo comando ]% ")
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

	var bandera bool = false // me ayudara para que despues de -> pueda concatenar negativos -12, porque concatena ->- y aparte el numero

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
			} else if examinarAsci == 95 { // _
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
				caracter = caracter + string(examinar) //cancatena
				analizador2(caracter)                  //ingresa en lista, en este caso guardara cuando sea ->
				caracter = ""                          //limpiara
				estado = 5                             // y mandara al estado 5 (donde verifica si vendra numero negativo) ej. ->-23
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
		case 5:
			//cuando 'bandera' sea falso entrara aca
			//caracter vendra vacio ""
			//entonces si viene - va concatenar
			//si viene numero despues de guion va concatenar y activar el 'bandera'
			//entonces cuando venga otro - no lo va concatenar porque 'bandera' es true y entrara en el else
			if examinarAsci == 45 && bandera == false { // -
				caracter = caracter + string(examinar)
				estado = 5
			} else if examinarAsci > 47 && examinarAsci < 58 { // numeros
				caracter = caracter + string(examinar)
				bandera = true
				estado = 5
			} else {
				if strings.Compare(caracter, "") != 0 { //esto es porque: viene vacio de estado 2, y como viene una letra, entra aca e ingresa un vacio a la lista
					analizador2(caracter)
				}
				caracter = ""
				bandera = false
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
		if strings.Compare(strings.ToLower(listaComandos[i]), "exec") == 0 {
			execComando(i)
		} else if strings.Compare(strings.ToLower(listaComandos[i]), "mkdisk") == 0 {
			mkdiskComando(i)
		} else if strings.Compare(strings.ToLower(listaComandos[i]), "rmdisk") == 0 {
			rmdiskComando(i)
		} else if strings.Compare(strings.ToLower(listaComandos[i]), "fdisk") == 0 {
			fdiskComando(i)
		}
	}
}

//--------------------------------INICIO EXEC-------------------------------//
//recibe el parametro index, que es el indice por donde actualmente se esta
func execComando(index int) {

	for i := index; i < len(listaComandos); i++ {
		if strings.Compare(strings.ToLower(listaComandos[i]), "path") == 0 { //cuando encuentre palabra reservada path
			if (strings.Compare(listaComandos[i-1], "-") == 0) && (strings.Compare(listaComandos[i+1], "->") == 0) { // validar si esta de esta forma -path->
				ruta := listaComandos[i+2]        //ruta
				if strings.Contains(ruta, "\"") { //si la ruta que viene contiene comillas
					ruta2 := ruta[1 : len(ruta)-1] //le quitamos comillas a la ruta
					leerArchivoExec(ruta2)         //funcion que leera el archivo
				} else { //sino tiene comillas manda la ruta normal
					leerArchivoExec(ruta)
				}
			} else {
				fmt.Print("\n[ ERROR: mal formato del comando 'exec' ]")
			}
		}
	}
}

//lee el texto que contiene un archivo
func leerArchivoExec(ruta string) {
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
					//imprimirListaComandos()
					logica()
					vaciarListaComandos()
					caracteres = ""
					fmt.Print("\n[ pause: presione 'enter' ]")
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
	//imprimirListaComandos()
	logica()
	vaciarListaComandos()
	caracteres = ""
}

//--------------------------------FIN EXEC----------------------------//

//-------------------------------INICIO MKDISK-------------------------------//
//MKDISK (crea disco duro)
func mkdiskComando(index int) {

	var size int = 0
	path := ""
	name := ""
	var unit string = "m"

	for i := index; i < len(listaComandos); i++ {

		if strings.Compare(strings.ToLower(listaComandos[i]), "size") == 0 {
			if (strings.Compare(listaComandos[i-1], "-") == 0) && (strings.Compare(listaComandos[i+1], "->") == 0) { // validar si esta de esta forma -size->
				tam, err := strconv.Atoi(listaComandos[i+2]) //convierto el valor a int
				size = tam
				if err != nil {
					fmt.Print("\n[ ERROR: Debe ingresar un numero en size de MKDISK ]")
				}

			} else {
				fmt.Print("\n[ ERROR: comando 'MKDISK' -> 'Size' ]")
			}

		} else if strings.Compare(strings.ToLower(listaComandos[i]), "path") == 0 { //cuando encuentre palabra reservada path
			if (strings.Compare(listaComandos[i-1], "-") == 0) && (strings.Compare(listaComandos[i+1], "->") == 0) { // validar si esta de esta forma -path->
				ruta := listaComandos[i+2]        //ruta
				if strings.Contains(ruta, "\"") { //si la ruta que viene contiene comillas
					ruta2 := ruta[1 : len(ruta)-1] //le quitamos comillas a la ruta
					path = ruta2                   //funcion que leera el archivo
				} else { //sino tiene comillas manda la ruta normal
					path = ruta
				}
			} else {
				fmt.Print("\n[ ERROR: comando 'MKDISK' -> 'path' ]")
			}
		} else if strings.Compare(strings.ToLower(listaComandos[i]), "name") == 0 {
			if (strings.Compare(listaComandos[i-1], "-") == 0) && (strings.Compare(listaComandos[i+1], "->") == 0) { // validar si esta de esta forma -name->
				name = listaComandos[i+2] //name
			} else {
				fmt.Print("\n[ ERROR: comando 'MKDISK' -> 'name' ]")
			}
		} else if strings.Compare(strings.ToLower(listaComandos[i]), "unit") == 0 {
			if (strings.Compare(listaComandos[i-1], "-") == 0) && (strings.Compare(listaComandos[i+1], "->") == 0) { // validar si esta de esta forma -name->
				unit = listaComandos[i+2] //toma el string
			} else {
				fmt.Print("\n[ ERROR:  Se ha producido un error con el comando 'MKDISK' -> 'unit' ]")
			}
		}
	}

	crearArchivo(size, path, name, unit)

}

//crea el archivo (disco)
func crearArchivo(size int, path string, name string, unit string) {

	//hacemos las operaciones para definir el tamanio
	if strings.Compare(strings.ToLower(unit), "k") == 0 {
		size = size * 1024
	} else if strings.Compare(strings.ToLower(unit), "m") == 0 {
		size = size * 1024 * 1024
	}

	ruta := path + name //concatenamos la ruta y el nombre del archivo
	file, err := os.Create(ruta)
	defer file.Close()
	if err != nil {
		log.Fatal(err)
	}

	//creamos una variable que sera un 0
	var otro int8 = 0
	s := &otro

	// --------------------------------DEFINIMOS EL TAMANIO DE NUESTRO ARCHIVO-------------------------------- //
	//Escribimos un 0 en el inicio del archivo
	var binario bytes.Buffer
	binary.Write(&binario, binary.BigEndian, s)
	escribirBytes(file, binario.Bytes())

	//Nos posicionamos en la ultima posicion
	file.Seek(int64(size-1), 0) // segundo parametro: 0, 1, 2.     0 -> Inicio, 1-> desde donde esta el puntero, 2 -> Del fin para atras
	//Escribimos un 0 al final del archivo.
	var binario2 bytes.Buffer
	binary.Write(&binario2, binary.BigEndian, s)
	escribirBytes(file, binario2.Bytes())

	//-------------------------------------------------------------------------------------------------------------//

	//Escribimos nuestro struct (MBR) en el inicio del archivo
	file.Seek(0, 0) // nos posicionamos en el inicio del archivo.

	//Asignamos valores a los atributos del struct.
	disco := mbr{}

	//tamanio disco
	disco.Tamanio = int64(size)

	t := time.Now()
	fecha := fmt.Sprintf("%d-%02d-%02dT%02d:%02d:%02d",
		t.Year(), t.Month(), t.Day(),
		t.Hour(), t.Minute(), t.Second())

	// Igualar cadenas a array de bytes (array de chars)
	copy(disco.Fecha[:], fecha)

	//numero de asignacion aleatorio
	disco.NumAsignacion = int64(rand.Intn(100))

	s1 := &disco

	//Escribimos struct (MBR)
	var binario3 bytes.Buffer
	binary.Write(&binario3, binary.BigEndian, s1)
	escribirBytes(file, binario3.Bytes())

}

//Método para escribir en un archivo
func escribirBytes(file *os.File, bytes []byte) {
	_, err := file.Write(bytes)

	if err != nil {
		log.Fatal(err)
	}
}

//-------------------------------FIN MKDISK-------------------------------//

//-------------------------------INICIO RMDISK-------------------------------//
//RMDISK (elimina un archivo)
func rmdiskComando(index int) {
	for i := index; i < len(listaComandos); i++ {
		if strings.Compare(strings.ToLower(listaComandos[i]), "path") == 0 { //cuando encuentre palabra reservada path
			if (strings.Compare(listaComandos[i-1], "-") == 0) && (strings.Compare(listaComandos[i+1], "->") == 0) { // validar si esta de esta forma -path->
				ruta := listaComandos[i+2]        //ruta
				if strings.Contains(ruta, "\"") { //si la ruta que viene contiene comillas
					ruta2 := ruta[1 : len(ruta)-1] //le quitamos comillas a la ruta
					os.Remove(ruta2)
				} else { //sino tiene comillas manda la ruta normal
					os.Remove(ruta)
				}
			} else {
				fmt.Print("\n[ ERROR: formato del comando 'rmdisk' ]")
			}
		}
	}
}

//-------------------------------FIN RMDISK-------------------------------//

//-------------------------------INICIO FDISK-------------------------------//
//FDISK (administra particiones del archivo, ajustes)
func fdiskComando(index int) {

	var size int64 = 0
	unit := "k"
	path := ""
	typee := "p"
	fit := "wf"
	deletee := ""
	name := ""
	add := 0

	for i := index; i < len(listaComandos); i++ {
		if strings.Compare(strings.ToLower(listaComandos[i]), "size") == 0 {
			if (strings.Compare(listaComandos[i-1], "-") == 0) && (strings.Compare(listaComandos[i+1], "->") == 0) { // validar si esta de esta forma -size->
				tam, err := strconv.Atoi(listaComandos[i+2]) //convierto el valor a int
				size = int64(tam)
				if err != nil {
					fmt.Print("\n[ ERROR: Debe ingresar un numero en 'size' de 'FDISK' ]")
				}

			} else {
				fmt.Print("\n[ ERROR: comando 'FDISK' -> 'size' ]")
			}

		} else if strings.Compare(strings.ToLower(listaComandos[i]), "unit") == 0 {
			if (strings.Compare(listaComandos[i-1], "-") == 0) && (strings.Compare(listaComandos[i+1], "->") == 0) { // validar si esta de esta forma -name->
				unit = listaComandos[i+2] //toma el string
			} else {
				fmt.Print("\n[ ERROR:  Se ha producido un error con el comando 'FDISK' -> 'unit' ]")
			}
		} else if strings.Compare(strings.ToLower(listaComandos[i]), "path") == 0 { //cuando encuentre palabra reservada path
			if (strings.Compare(listaComandos[i-1], "-") == 0) && (strings.Compare(listaComandos[i+1], "->") == 0) { // validar si esta de esta forma -path->
				ruta := listaComandos[i+2]        //ruta
				if strings.Contains(ruta, "\"") { //si la ruta que viene contiene comillas
					ruta2 := ruta[1 : len(ruta)-1] //le quitamos comillas a la ruta
					path = ruta2                   //funcion que leera el archivo
				} else { //sino tiene comillas manda la ruta normal
					path = ruta
				}
			} else {
				fmt.Print("\n[ ERROR: comando 'FDISK' -> 'path' ]")
			}
		} else if strings.Compare(strings.ToLower(listaComandos[i]), "type") == 0 {
			if (strings.Compare(listaComandos[i-1], "-") == 0) && (strings.Compare(listaComandos[i+1], "->") == 0) { // validar si esta de esta forma -name->
				typee = listaComandos[i+2] //name
			} else {
				fmt.Print("\n[ ERROR: comando 'FDISK' -> 'type' ]")
			}
		} else if strings.Compare(strings.ToLower(listaComandos[i]), "fit") == 0 {
			if (strings.Compare(listaComandos[i-1], "-") == 0) && (strings.Compare(listaComandos[i+1], "->") == 0) { // validar si esta de esta forma -name->
				fit = listaComandos[i+2] //name
			} else {
				fmt.Print("\n[ ERROR: comando 'FDISK' -> 'fit' ]")
			}
		} else if strings.Compare(strings.ToLower(listaComandos[i]), "delete") == 0 {
			if (strings.Compare(listaComandos[i-1], "-") == 0) && (strings.Compare(listaComandos[i+1], "->") == 0) { // validar si esta de esta forma -name->
				deletee = listaComandos[i+2] //name
			} else {
				fmt.Print("\n[ ERROR: comando 'FDISK' -> 'delete' ]")
			}
		} else if strings.Compare(strings.ToLower(listaComandos[i]), "name") == 0 {
			if (strings.Compare(listaComandos[i-1], "-") == 0) && (strings.Compare(listaComandos[i+1], "->") == 0) { // validar si esta de esta forma -name->
				name = listaComandos[i+2] //name
			} else {
				fmt.Print("\n[ ERROR: comando 'FDISK' -> 'name' ]")
			}
		} else if strings.Compare(strings.ToLower(listaComandos[i]), "add") == 0 {
			if (strings.Compare(listaComandos[i-1], "-") == 0) && (strings.Compare(listaComandos[i+1], "->") == 0) { // validar si esta de esta forma -name->
				tam, err := strconv.Atoi(listaComandos[i+2]) //convierto el valor a int
				add = tam
				if err != nil {
					fmt.Print("\n[ ERROR: Debe ingresar un numero en 'size' de 'FDISK' ]")
				}
			} else {
				fmt.Print("\n[ ERROR: comando 'FDISK' -> 'add' ]")
			}
		}
	}
	operacionFdisk(size, unit, path, typee, fit, deletee, name, add)
}

func operacionFdisk(size int64, unit string, path string, typee string, fit string, deletee string, name string, add int) {

	//si es delete
	if strings.Compare(deletee, "") != 0 { //si hay que eliminar una particion
		//si existe particion primaria o extendida con ese nombre
		if validarSiExisteParticionPrimariaExtendidaConNombreEspecifico(path, name) {
			eliminarParticionPrimariaExtendida(path, name)
			//si existe particion logica con ese nombre
		} else if validarSiExisteParticionLogicaConNombreEspecifico(path, name) {
			eliminarParticionLogica(path, name)
		} else {
			fmt.Print("\n[ ERROR: no exite particion con nombre: ", name, " ]")
		}

		//si es add
	} else if add != 0 { //agregar o quitar espacio de particion
		//si existe particion primaria o extendida con ese nombre
		if validarSiExisteParticionPrimariaExtendidaConNombreEspecifico(path, name) {
			if validaSiSeLePuedeaddEspacioEnPrimariaExtendida(path, name, unit, int64(add)) {
				//agrega o quita espacio en extendida o primaria
				addEspacioEnPrimariaExtendida(path, name, unit, int64(add))
			} else {
				fmt.Print("\n[ ERROR: no se puede agregar o quitar espacio en particion con nombre: ", name, " ]")
			}

			//si existe particion logica con ese nombre
		} else if validarSiExisteParticionLogicaConNombreEspecifico(path, name) {
		} else {
			fmt.Print("\n[ ERROR: no exite particion con nombre: ", name, " ]")
		}

		//crear paarticion
	} else { //crea una particion
		agregarParticion(size, unit, path, typee, fit, name)
	}
}

func agregarParticion(size int64, unit string, path string, typee string, fit string, name string) {

	//esto es porque hay una validacion de: 4 primarias, o 3 primarias y una extendida
	if (strings.Compare(strings.ToLower(typee), "p") == 0) || (strings.Compare(strings.ToLower(typee), "e") == 0) {
		if validarLimiteDeParticionesEnDisco(path) { //si se puede agregar otra particion
			//si es primaria
			if strings.Compare(strings.ToLower(typee), "p") == 0 {
				//si el disco aun tiene espacio
				if validarQueTengaEspacioElDisco(path, size, unit) {
					//inserta particion primaria
					insertarParticionPrimaria(path, size, typee, fit, name, unit)
				} else {
					fmt.Print("\n[ ERROR: no hay espacio para agregar la particion primaria: ", name, " ]")
				}
				//si es extendida
			} else if strings.Compare(strings.ToLower(typee), "e") == 0 {
				//si aun no existe una extendida
				if validarSiExisteParticionExtendida(path) == false {
					//si el disco aun tiene espacio
					if validarQueTengaEspacioElDisco(path, size, unit) {
						//inserta particion extendida
						insertarParticionExtendida(path, size, typee, fit, name, unit)
					} else {
						fmt.Print("\n[ ERROR: no hay espacio para agregar la particion extendida: ", name, " ]")
					}

				} else {
					fmt.Print("\n[ ERROR: ya existe particion extendida en el disco, no se puede agregar: ", name, " ]")
				}
			}
		} else {
			fmt.Print("\n[ ERROR: Ya alcanzo el limite de de particiones en el disco: ]")
		}
		//si es logica
	} else if strings.Compare(strings.ToLower(typee), "l") == 0 {
		//si existe particion extendida
		if validarSiExisteParticionExtendida(path) {
			//si hay espacio dentro de la extendida
			if validarQueTengaEspacioParticionExtendida(path, size, unit) {
				//inserta particion logica
				insertarParticionLogica(path, size, typee, fit, name, unit)
			} else {
				fmt.Print("\n[ ERROR: no hay espacio para agregar la particion logica: ", name, " ]")
			}
		} else {
			fmt.Print("\n[ ERROR: no existe particion extendida en el disco, no se puede agregar: ", name, " ]")
		}
	}
}

//valida ya que el disco puede tener maximo 4 particiones
func validarLimiteDeParticionesEnDisco(path string) bool {
	//Abrimos/creamos un archivo.
	file, err := os.OpenFile(path, os.O_RDWR, 0644)
	defer file.Close()
	if err != nil { //validar que no sea nulo.
		log.Fatal(err)
	}

	//Declaramos variable de tipo mbr
	m := mbr{}

	//Obtenemos el tamanio del mbr
	var size int = int(unsafe.Sizeof(m))

	//Lee la cantidad de <size> bytes del archivo
	data := leerBytesFdisk(file, size)

	//Convierte la data en un buffer,necesario para decodificar binario
	buffer := bytes.NewBuffer(data)

	//Decodificamos y guardamos en la variable m
	err = binary.Read(buffer, binary.BigEndian, &m)
	if err != nil {
		log.Fatal("binary.Read failed", err)
	}

	//obtengo el arreglo de particiones
	misParticiones := m.Particiones

	contador := 0

	//recorro para ver cuantos espacios vacios hay
	for i := 0; i < 4; i++ {
		actual := misParticiones[i]
		if actual.Tamanio == 0 {
			contador++
		}
	}

	//si hay algun espacio para otra particion
	if contador > 0 {
		return true //retorna que hay espacio
	}

	return false
}

//valida si hay un espacio en el mbr para particion primaria
func validarQueTengaEspacioElDisco(path string, sizeParticion int64, unit string) bool {

	//se hace la convercion de kb a bytes, o mb a bytes, segun sea el caso
	if strings.Compare(strings.ToLower(unit), "k") == 0 {
		sizeParticion = sizeParticion * 1024
	} else if strings.Compare(strings.ToLower(unit), "m") == 0 {
		sizeParticion = sizeParticion * 1024 * 1024
	}

	//Abrimos/creamos un archivo.
	file, err := os.OpenFile(path, os.O_RDWR, 0644)
	defer file.Close()
	if err != nil { //validar que no sea nulo.
		log.Fatal(err)
	}

	//Declaramos variable de tipo mbr
	m := mbr{}

	//Obtenemos el tamanio del mbr
	var size int = int(unsafe.Sizeof(m))

	//Lee la cantidad de <size> bytes del archivo
	data := leerBytesFdisk(file, size)

	//Convierte la data en un buffer,necesario para decodificar binario
	buffer := bytes.NewBuffer(data)

	//Decodificamos y guardamos en la variable m
	err = binary.Read(buffer, binary.BigEndian, &m)
	if err != nil {
		log.Fatal("binary.Read failed", err)
	}

	//obtengo el arreglo de particiones
	misParticiones := m.Particiones

	//sera la sumatoria de byte de todas las particiones
	var contadorSize int64 = 0

	//recorro para sumar todos los byte de las particiones
	for i := 0; i < 4; i++ {
		actual := misParticiones[i]
		if actual.Tamanio != 0 {
			contadorSize = contadorSize + int64(actual.Tamanio)
		}
	}

	//espacio disponible = (tamanio disco) - (espacio actual de todas las particiones)
	var espacioDisponible int64 = int64(m.Tamanio) - contadorSize

	//si hay espacio aun
	if sizeParticion <= espacioDisponible {
		return true //retorna que hay espacio
	}

	return false
}

//inserta particion primaria en el disco
func insertarParticionPrimaria(path string, sizePart int64, typee string, fit string, name string, unit string) {

	//se hace la convercion de kb a bytes, o mb a bytes, segun sea el caso
	if strings.Compare(strings.ToLower(unit), "k") == 0 {
		sizePart = sizePart * 1024
	} else if strings.Compare(strings.ToLower(unit), "m") == 0 {
		sizePart = sizePart * 1024 * 1024
	}

	//Abrimos/creamos un archivo.
	file, err := os.OpenFile(path, os.O_RDWR, 0644)
	defer file.Close()
	if err != nil { //validar que no sea nulo.
		log.Fatal(err)
	}

	//Declaramos variable de tipo mbr
	m := mbr{}

	//Obtenemos el tamanio del mbr
	var size int = int(unsafe.Sizeof(m))

	//Lee la cantidad de <size> bytes del archivo
	data := leerBytesFdisk(file, size)

	//Convierte la data en un buffer,necesario para decodificar binario
	buffer := bytes.NewBuffer(data)

	//Decodificamos y guardamos en la variable m
	err = binary.Read(buffer, binary.BigEndian, &m)
	if err != nil {
		log.Fatal("binary.Read failed", err)
	}

	//accedo a las particiones
	misParticiones := m.Particiones

	//para ver la posicion vacia
	contador := 0

	//recorro para ver cuual esta vacia
	for i := 0; i < 4; i++ {
		actual := misParticiones[i]
		if actual.Tamanio == 0 {
			contador = i
			break
		}
	}

	//se inserta despues del MBR
	if contador == 0 {
		//creo particion primaria
		particionPrimariaNew := NodoParticion{}

		//agrego atributos a particion primaria
		copy(particionPrimariaNew.Name[:], name)
		particionPrimariaNew.Tamanio = sizePart
		particionPrimariaNew.Estado = 1
		particionPrimariaNew.TipoParticion = typee[0]
		copy(particionPrimariaNew.TipoAjuste[:], fit)
		particionPrimariaNew.Start = int64(size) + 1

		//inserto particion
		misParticiones[contador] = particionPrimariaNew

		//pueden ser en la posicion 1, 2, 3
	} else if contador > 0 {
		//creo particion primaria
		particionPrimariaNew := NodoParticion{}

		//agrego atributos a particion primaria
		copy(particionPrimariaNew.Name[:], name)
		particionPrimariaNew.Tamanio = sizePart
		particionPrimariaNew.Estado = 1
		particionPrimariaNew.TipoParticion = typee[0]
		copy(particionPrimariaNew.TipoAjuste[:], fit)

		//Donde empieza? empieza donde termina la particion anterior
		inicioPartAnt := misParticiones[contador-1].Start //byte donde inicia la particion anterior
		tamPartAnt := misParticiones[contador-1].Tamanio  //tamanio de la particion anterior
		finPartAnt := inicioPartAnt + tamPartAnt          //byte donde finaliza la particion anterior
		particionPrimariaNew.Start = finPartAnt + 1       //aqui empieza la nueva particion

		//inserto particion
		misParticiones[contador] = particionPrimariaNew
	}

	fmt.Println("\nINSERTO PRIMARIA:")
	fmt.Println("	arr pos 0 Tamanio : ", misParticiones[0].Tamanio, " Star: ", misParticiones[0].Start, " Tipo: ", string(misParticiones[0].TipoParticion))
	fmt.Println("	arr pos 1 Tamanio : ", misParticiones[1].Tamanio, " Star: ", misParticiones[1].Start, " Tipo: ", string(misParticiones[1].TipoParticion))
	fmt.Println("	arr pos 2 Tamanio : ", misParticiones[2].Tamanio, " Star: ", misParticiones[2].Start, " Tipo: ", string(misParticiones[2].TipoParticion))
	fmt.Println("	arr pos 3 Tamanio : ", misParticiones[3].Tamanio, " Star: ", misParticiones[3].Start, " Tipo: ", string(misParticiones[3].TipoParticion))

	//las particiones actuales en el disco se encuentran en 'm.particiones'
	//cuando se creo una nueva particion se agregadron a 'misPartiiones'
	//entonces 'misParticiones' tiene las actuales, mas la nueva que se le inserto
	//por eso se iguala de nuevo, para que em 'm.Particiones' se guarden particiones ya actualizadas
	m.Particiones = misParticiones

	file.Seek(0, 0)
	s1 := &m

	//Reescribimos struct (MBR)
	var binario3 bytes.Buffer
	binary.Write(&binario3, binary.BigEndian, s1)
	escribirBytes(file, binario3.Bytes())

}

//valida si hay una particion extendida en el disco
func validarSiExisteParticionExtendida(path string) bool {
	//Abrimos/creamos un archivo.
	file, err := os.OpenFile(path, os.O_RDWR, 0644)
	defer file.Close()
	if err != nil { //validar que no sea nulo.
		log.Fatal(err)
	}

	//Declaramos variable de tipo mbr
	m := mbr{}

	//Obtenemos el tamanio del mbr
	var size int = int(unsafe.Sizeof(m))

	//Lee la cantidad de <size> bytes del archivo
	data := leerBytesFdisk(file, size)

	//Convierte la data en un buffer,necesario para decodificar binario
	buffer := bytes.NewBuffer(data)

	//Decodificamos y guardamos en la variable m
	err = binary.Read(buffer, binary.BigEndian, &m)
	if err != nil {
		log.Fatal("binary.Read failed", err)
	}

	//obtengo el arreglo de particiones
	misParticiones := m.Particiones

	existe := false

	//recorro para ver cuantos espacios vacios hay
	for i := 0; i < 4; i++ {
		actual := misParticiones[i]
		if strings.Compare(strings.ToLower(string(actual.TipoParticion)), "e") == 0 {
			existe = true
			break
		}
	}

	//si ya existe una extendida
	if existe {
		return true
	}

	return false
}

//inserta particion extendida en el disco
func insertarParticionExtendida(path string, sizePart int64, typee string, fit string, name string, unit string) {
	//se hace la convercion de kb a bytes, o mb a bytes, segun sea el caso
	if strings.Compare(strings.ToLower(unit), "k") == 0 {
		sizePart = sizePart * 1024
	} else if strings.Compare(strings.ToLower(unit), "m") == 0 {
		sizePart = sizePart * 1024 * 1024
	}

	//Abrimos/creamos un archivo.
	file, err := os.OpenFile(path, os.O_RDWR, 0644)
	defer file.Close()
	if err != nil { //validar que no sea nulo.
		log.Fatal(err)
	}

	//Declaramos variable de tipo mbr
	m := mbr{}

	//Obtenemos el tamanio del mbr
	var size int = int(unsafe.Sizeof(m))

	//Lee la cantidad de <size> bytes del archivo
	data := leerBytesFdisk(file, size)

	//Convierte la data en un buffer,necesario para decodificar binario
	buffer := bytes.NewBuffer(data)

	//Decodificamos y guardamos en la variable m
	err = binary.Read(buffer, binary.BigEndian, &m)
	if err != nil {
		log.Fatal("binary.Read failed", err)
	}

	//accedo a las particiones
	misParticiones := m.Particiones

	//para ver la posicion vacia
	contador := 0

	//recorro para ver cuual esta vacia
	for i := 0; i < 4; i++ {
		actual := misParticiones[i]
		if actual.Tamanio == 0 {
			contador = i
			break
		}
	}

	//se inserta despues del MBR
	if contador == 0 {
		//creo particion primaria
		particionPrimariaNew := NodoParticion{}

		//agrego atributos a particion primaria
		copy(particionPrimariaNew.Name[:], name)
		particionPrimariaNew.Tamanio = sizePart
		particionPrimariaNew.Estado = 1
		particionPrimariaNew.TipoParticion = typee[0]
		copy(particionPrimariaNew.TipoAjuste[:], fit)
		particionPrimariaNew.Start = int64(size) + 1

		//inserto particion
		misParticiones[contador] = particionPrimariaNew

		//pueden ser en la posicion 1, 2, 3
	} else if contador > 0 {
		//creo particion primaria
		particionPrimariaNew := NodoParticion{}

		//agrego atributos a particion primaria
		copy(particionPrimariaNew.Name[:], name)
		particionPrimariaNew.Tamanio = sizePart
		particionPrimariaNew.Estado = 1
		particionPrimariaNew.TipoParticion = typee[0]
		copy(particionPrimariaNew.TipoAjuste[:], fit)

		//Donde empieza? empieza donde termina la particion anterior
		inicioPartAnt := misParticiones[contador-1].Start //byte donde inicia la particion anterior
		tamPartAnt := misParticiones[contador-1].Tamanio  //tamanio de la particion anterior
		finPartAnt := inicioPartAnt + tamPartAnt          //byte donde finaliza la particion anterior
		particionPrimariaNew.Start = finPartAnt + 1       //aqui empieza la nueva particion

		//inserto particion
		misParticiones[contador] = particionPrimariaNew
	}

	fmt.Println("\nINSERTO EXTENDIDA:")
	fmt.Println("	arr pos 0 Tamanio : ", misParticiones[0].Tamanio, " Star: ", misParticiones[0].Start, " Tipo: ", string(misParticiones[0].TipoParticion))
	fmt.Println("	arr pos 1 Tamanio : ", misParticiones[1].Tamanio, " Star: ", misParticiones[1].Start, " Tipo: ", string(misParticiones[1].TipoParticion))
	fmt.Println("	arr pos 2 Tamanio : ", misParticiones[2].Tamanio, " Star: ", misParticiones[2].Start, " Tipo: ", string(misParticiones[2].TipoParticion))
	fmt.Println("	arr pos 3 Tamanio : ", misParticiones[3].Tamanio, " Star: ", misParticiones[3].Start, " Tipo: ", string(misParticiones[3].TipoParticion))

	//las particiones actuales en el disco se encuentran en 'm.particiones'
	//cuando se creo una nueva particion se agregadron a 'misPartiiones'
	//entonces 'misParticiones' tiene las actuales, mas la nueva que se le inserto
	//por eso se iguala de nuevo, para que em 'm.Particiones' se guarden particiones ya actualizadas
	m.Particiones = misParticiones

	file.Seek(0, 0)
	s1 := &m

	//Reescribimos struct (MBR)
	var binario3 bytes.Buffer
	binary.Write(&binario3, binary.BigEndian, s1)
	escribirBytes(file, binario3.Bytes())
}

//valida si hay espacio en la particion extendida, dentro del disco
func validarQueTengaEspacioParticionExtendida(path string, sizeParticion int64, unit string) bool {

	//se hace la convercion de kb a bytes, o mb a bytes, segun sea el caso
	if strings.Compare(strings.ToLower(unit), "k") == 0 {
		sizeParticion = sizeParticion * 1024
	} else if strings.Compare(strings.ToLower(unit), "m") == 0 {
		sizeParticion = sizeParticion * 1024 * 1024
	}

	//Abrimos/creamos un archivo.
	file, err := os.OpenFile(path, os.O_RDWR, 0644)
	defer file.Close()
	if err != nil { //validar que no sea nulo.
		log.Fatal(err)
	}

	//Declaramos variable de tipo mbr
	m := mbr{}

	//Obtenemos el tamanio del mbr
	var size int = int(unsafe.Sizeof(m))

	//Lee la cantidad de <size> bytes del archivo
	data := leerBytesFdisk(file, size)

	//Convierte la data en un buffer,necesario para decodificar binario
	buffer := bytes.NewBuffer(data)

	//Decodificamos y guardamos en la variable m
	err = binary.Read(buffer, binary.BigEndian, &m)
	if err != nil {
		log.Fatal("binary.Read failed", err)
	}

	//obtengo el arreglo de particiones
	misParticiones := m.Particiones

	//obtengo el indice donde se encuentra la particion extendida
	posicionExtendida := 0
	for i := 0; i < 4; i++ {
		actual := misParticiones[i]
		if strings.Compare(strings.ToLower(string(actual.TipoParticion)), "e") == 0 {
			posicionExtendida = i
			break
		}
	}

	misParticionesLogicas := misParticiones[posicionExtendida].ParticionesLogicas

	//sera la sumatoria de byte de todas las particiones primarias
	var contadorSize int64 = 0

	//recorro para sumar todos los byte de las particiones
	for i := 0; i < 5; i++ {
		actual := misParticionesLogicas[i]
		if actual.Tamanio != 0 {
			contadorSize = contadorSize + int64(actual.Tamanio)
		}
	}

	//espacio disponible = (tamanio particion extendida) - (espacio actual de todas las particiones logicas)
	var espacioDisponible int64 = int64(misParticiones[posicionExtendida].Tamanio) - contadorSize

	//si hay espacio aun
	if sizeParticion <= espacioDisponible {
		return true //retorna que hay espacio
	}

	return false
}

//inserta particion logica, dentro de la extendida
func insertarParticionLogica(path string, sizePart int64, typee string, fit string, name string, unit string) {
	//se hace la convercion de kb a bytes, o mb a bytes, segun sea el caso
	if strings.Compare(strings.ToLower(unit), "k") == 0 {
		sizePart = sizePart * 1024
	} else if strings.Compare(strings.ToLower(unit), "m") == 0 {
		sizePart = sizePart * 1024 * 1024
	}

	//Abrimos/creamos un archivo.
	file, err := os.OpenFile(path, os.O_RDWR, 0644)
	defer file.Close()
	if err != nil { //validar que no sea nulo.
		log.Fatal(err)
	}

	//Declaramos variable de tipo mbr
	m := mbr{}

	//Obtenemos el tamanio del mbr
	var size int = int(unsafe.Sizeof(m))

	//Lee la cantidad de <size> bytes del archivo
	data := leerBytesFdisk(file, size)

	//Convierte la data en un buffer,necesario para decodificar binario
	buffer := bytes.NewBuffer(data)

	//Decodificamos y guardamos en la variable m
	err = binary.Read(buffer, binary.BigEndian, &m)
	if err != nil {
		log.Fatal("binary.Read failed", err)
	}

	//accedo a las particiones
	misParticiones := m.Particiones

	//obtengo el indice donde se encuentra la particion extendida
	posicionExtendida := 0
	for i := 0; i < 4; i++ {
		actual := misParticiones[i]
		if strings.Compare(strings.ToLower(string(actual.TipoParticion)), "e") == 0 {
			posicionExtendida = i
			break
		}
	}

	//accedo a las particiones logicas, quee estan dentro de la extendida
	misParticionesLogicas := misParticiones[posicionExtendida].ParticionesLogicas

	//para ver la posicion logica vacia
	posicionLogicaVacia := 0

	//recorro para ver cuual esta vacia
	for i := 0; i < 5; i++ {
		actual := misParticionesLogicas[i]
		if actual.Tamanio == 0 {
			posicionLogicaVacia = i
			break
		}
	}

	//se inserta despues del MBR
	if posicionLogicaVacia == 0 {

		//buscando particion siguiente
		posSiguiente := -1
		for i := posicionLogicaVacia + 1; i < 5; i++ { //empieza a buscar desde la posicion 1, y que insertara en la posicion 0
			//si encuantra despues una particion, guardo posicion donde la encuentra
			if misParticionesLogicas[i].Tamanio != 0 {
				posSiguiente = i
				break
			}
		}

		//CASO 1: no tenga siguiente
		if posSiguiente == -1 {

			//creo particion primaria
			particionLogicaNew := NodoParticionLogica{}

			//agrego atributos a particion primaria
			copy(particionLogicaNew.Name[:], name)
			particionLogicaNew.Tamanio = sizePart
			particionLogicaNew.Estado = 1
			particionLogicaNew.TipoParticion = typee[0]
			copy(particionLogicaNew.TipoAjuste[:], fit)
			particionLogicaNew.Start = int64(misParticiones[posicionExtendida].Start) + 1 //inicia donde inicia la extend
			particionLogicaNew.Next = int64(-1)

			//inserto particion
			misParticionesLogicas[posicionLogicaVacia] = particionLogicaNew
			fmt.Println("Inicio: ", name, " : ", particionLogicaNew.Start)

			//CASO 2: tenga siguiente
		} else if posSiguiente != -1 {

			//creo particion primaria
			particionLogicaNew := NodoParticionLogica{}

			//agrego atributos a particion primaria
			copy(particionLogicaNew.Name[:], name)
			particionLogicaNew.Tamanio = sizePart
			particionLogicaNew.Estado = 1
			particionLogicaNew.TipoParticion = typee[0]
			copy(particionLogicaNew.TipoAjuste[:], fit)
			particionLogicaNew.Start = int64(misParticiones[posicionExtendida].Start) + 1 //inicia donde inicia la extend
			particionLogicaNew.Next = misParticionesLogicas[posSiguiente].Start

			//inserto particion
			misParticionesLogicas[posicionLogicaVacia] = particionLogicaNew
			fmt.Println("Inicio: ", name, " : ", particionLogicaNew.Start)

		}

		//pueden ser en la posicion 1, 2, 3
	} else if posicionLogicaVacia > 0 {

		//buscando particion anterior
		posAnterior := -1
		for i := posicionLogicaVacia - 1; i > -1; i-- { //empieza a buscar una antes de la que se va eliminar
			//si encuantra antes una particion, guardo posicion donde la encuentra
			if misParticionesLogicas[i].Tamanio != 0 {
				posAnterior = i
				break
			}
		}

		//buscando particion siguiente
		posSiguiente := -1
		for i := posicionLogicaVacia + 1; i < 5; i++ { //empieza a buscar una despues de la que se va eliminar
			//si encuantra despues una particion, guardo posicion donde la encuentra
			if misParticionesLogicas[i].Tamanio != 0 {
				posSiguiente = i
				break
			}
		}

		//CASO 1: tenga anterior y no tenga siguiente
		if (posAnterior != -1) && (posSiguiente == -1) {

			//creo particion primaria
			particionPrimariaNew := NodoParticionLogica{}

			//agrego atributos a particion primaria
			copy(particionPrimariaNew.Name[:], name)
			particionPrimariaNew.Tamanio = sizePart
			particionPrimariaNew.Estado = 1
			particionPrimariaNew.TipoParticion = typee[0]
			copy(particionPrimariaNew.TipoAjuste[:], fit)

			//Donde empieza? empieza donde termina la particion anterior
			inicioPartAnt := misParticionesLogicas[posicionLogicaVacia-1].Start //byte donde inicia la particion anterior
			tamPartAnt := misParticionesLogicas[posicionLogicaVacia-1].Tamanio  //tamanio de la particion anterior
			finPartAnt := inicioPartAnt + tamPartAnt                            //byte donde finaliza la particion anterior
			particionPrimariaNew.Start = finPartAnt + 1                         //aqui empieza la nueva particion

			//next anterior = star nuevo
			//next nuevo = -1
			misParticionesLogicas[posAnterior].Next = particionPrimariaNew.Start
			particionPrimariaNew.Next = int64(-1)

			//inserto particion
			misParticionesLogicas[posicionLogicaVacia] = particionPrimariaNew
			fmt.Println("Inicio: ", name, " : ", particionPrimariaNew.Start)

			//CASO 2: tenga anterior y siguiente
		} else if (posAnterior != -1) && (posSiguiente != -1) {

			//creo particion primaria
			particionPrimariaNew := NodoParticionLogica{}

			//agrego atributos a particion primaria
			copy(particionPrimariaNew.Name[:], name)
			particionPrimariaNew.Tamanio = sizePart
			particionPrimariaNew.Estado = 1
			particionPrimariaNew.TipoParticion = typee[0]
			copy(particionPrimariaNew.TipoAjuste[:], fit)

			//Donde empieza? empieza donde termina la particion anterior
			inicioPartAnt := misParticionesLogicas[posicionLogicaVacia-1].Start //byte donde inicia la particion anterior
			tamPartAnt := misParticionesLogicas[posicionLogicaVacia-1].Tamanio  //tamanio de la particion anterior
			finPartAnt := inicioPartAnt + tamPartAnt                            //byte donde finaliza la particion anterior
			particionPrimariaNew.Start = finPartAnt + 1                         //aqui empieza la nueva particion

			//next nuevo = next anterior
			//next anterior = star nuevo
			particionPrimariaNew.Next = misParticionesLogicas[posAnterior].Next
			misParticionesLogicas[posAnterior].Next = particionPrimariaNew.Start

			//inserto particion
			misParticionesLogicas[posicionLogicaVacia] = particionPrimariaNew
			fmt.Println("Inicio: ", name, " : ", particionPrimariaNew.Start)
		}

	}

	fmt.Println("\nINSERTO LOGICA:")
	fmt.Println("	arr pos 0 Tamanio : ", misParticionesLogicas[0].Tamanio, " Star: ", misParticionesLogicas[0].Start, " Next: ", misParticionesLogicas[0].Next, " Tipo: ", string(misParticionesLogicas[0].TipoParticion))
	fmt.Println("	arr pos 1 Tamanio : ", misParticionesLogicas[1].Tamanio, " Star: ", misParticionesLogicas[1].Start, " Next: ", misParticionesLogicas[1].Next, " Tipo: ", string(misParticionesLogicas[1].TipoParticion))
	fmt.Println("	arr pos 2 Tamanio : ", misParticionesLogicas[2].Tamanio, " Star: ", misParticionesLogicas[2].Start, " Next: ", misParticionesLogicas[2].Next, " Tipo: ", string(misParticionesLogicas[2].TipoParticion))
	fmt.Println("	arr pos 3 Tamanio : ", misParticionesLogicas[3].Tamanio, " Star: ", misParticionesLogicas[3].Start, " Next: ", misParticionesLogicas[3].Next, " Tipo: ", string(misParticionesLogicas[3].TipoParticion))
	fmt.Println("	arr pos 4 Tamanio : ", misParticionesLogicas[4].Tamanio, " Star: ", misParticionesLogicas[4].Start, " Next: ", misParticionesLogicas[4].Next, " Tipo: ", string(misParticionesLogicas[4].TipoParticion))

	//las particiones logicas actuales se encuentran en 'misParticiones[posicionExtendida].ParticionesLogicas'
	//cuando se crea una nueva particion logica se agregan a 'misParticionesPrimarias'
	//entonces 'misParticionesPrimarias' tienen las actuales, mas la nueva que se inserto
	//por eso se iguala de nuevo, para que 'misParticiones[posicionExtendida].ParticionesLogicas', e guarden particiones ya actualizadas
	misParticiones[posicionExtendida].ParticionesLogicas = misParticionesLogicas

	//para que se actualice nada mas
	m.Particiones = misParticiones

	file.Seek(0, 0)
	s1 := &m

	//Reescribimos struct (MBR)
	var binario3 bytes.Buffer
	binary.Write(&binario3, binary.BigEndian, s1)
	escribirBytes(file, binario3.Bytes())
}

//validar si existe particion primaria o extendida dado nombre (para delete)
func validarSiExisteParticionPrimariaExtendidaConNombreEspecifico(path string, name string) bool {
	//Abrimos/creamos un archivo.
	file, err := os.OpenFile(path, os.O_RDWR, 0644)
	defer file.Close()
	if err != nil { //validar que no sea nulo.
		log.Fatal(err)
	}

	//Declaramos variable de tipo mbr
	m := mbr{}

	//Obtenemos el tamanio del mbr
	var size int = int(unsafe.Sizeof(m))

	//Lee la cantidad de <size> bytes del archivo
	data := leerBytesFdisk(file, size)

	//Convierte la data en un buffer,necesario para decodificar binario
	buffer := bytes.NewBuffer(data)

	//Decodificamos y guardamos en la variable m
	err = binary.Read(buffer, binary.BigEndian, &m)
	if err != nil {
		log.Fatal("binary.Read failed", err)
	}

	//obtengo el arreglo de particiones
	misParticiones := m.Particiones

	existe := false

	//recorro para ver si existe nombre
	for i := 0; i < 4; i++ {
		actual := misParticiones[i]

		//eliminando espacios en blanco o nulos del name
		nombrePart := ""
		for j := 0; j < 16; j++ {
			if actual.Name[j] != 0 {
				nombrePart += string(actual.Name[j])
			}
		}
		if strings.Compare(strings.ToLower(nombrePart), strings.ToLower(name)) == 0 {
			existe = true
			break
		}
	}

	//si existe particion con ese nombre
	if existe {
		return true
	}

	return false

}

//eliminar particion primaria o extendida
func eliminarParticionPrimariaExtendida(path string, name string) {
	//Abrimos/creamos un archivo.
	file, err := os.OpenFile(path, os.O_RDWR, 0644)
	defer file.Close()
	if err != nil { //validar que no sea nulo.
		log.Fatal(err)
	}

	//Declaramos variable de tipo mbr
	m := mbr{}

	//Obtenemos el tamanio del mbr
	var size int = int(unsafe.Sizeof(m))

	//Lee la cantidad de <size> bytes del archivo
	data := leerBytesFdisk(file, size)

	//Convierte la data en un buffer,necesario para decodificar binario
	buffer := bytes.NewBuffer(data)

	//Decodificamos y guardamos en la variable m
	err = binary.Read(buffer, binary.BigEndian, &m)
	if err != nil {
		log.Fatal("binary.Read failed", err)
	}

	//obtengo el arreglo de particiones
	misParticiones := m.Particiones

	posicionParticion := 0

	//recorro para ver cuantos espacios vacios hay
	for i := 0; i < 4; i++ {
		actual := misParticiones[i]

		//eliminando espacios en blanco o nulos del name
		nombrePart := ""
		for j := 0; j < 16; j++ {
			if actual.Name[j] != 0 {
				nombrePart += string(actual.Name[j])
			}
		}
		if strings.Compare(strings.ToLower(nombrePart), strings.ToLower(name)) == 0 {
			posicionParticion = i
			break
		}
	}

	//creo una particion vacia
	particionNew := NodoParticion{}

	//inserto particion vacia en la posicion a eliminar
	misParticiones[posicionParticion] = particionNew

	fmt.Println("\nDELETE PRIMARIA O EXTENDIDA:")
	fmt.Println("	arr pos 0 Tamanio : ", misParticiones[0].Tamanio, " Tipo: ", string(misParticiones[0].TipoParticion))
	fmt.Println("	arr pos 1 Tamanio : ", misParticiones[1].Tamanio, " Tipo: ", string(misParticiones[1].TipoParticion))
	fmt.Println("	arr pos 2 Tamanio : ", misParticiones[2].Tamanio, " Tipo: ", string(misParticiones[2].TipoParticion))
	fmt.Println("	arr pos 3 Tamanio : ", misParticiones[3].Tamanio, " Tipo: ", string(misParticiones[3].TipoParticion))

	//las particiones actuales en el disco se encuentran en 'm.particiones'
	//cuando se elimino una particion se elimino de 'misPartiiones'
	//entonces 'misParticiones' tiene las actuales, con la que se acaba de liminar
	//por eso se iguala de nuevo, para que em 'm.Particiones' ya no aparezca la que se elimino
	m.Particiones = misParticiones

	file.Seek(0, 0)
	s1 := &m

	//Reescribimos struct (MBR)
	var binario3 bytes.Buffer
	binary.Write(&binario3, binary.BigEndian, s1)
	escribirBytes(file, binario3.Bytes())

}

//func validar existe particion logica dado nombre (para delete)
func validarSiExisteParticionLogicaConNombreEspecifico(path string, name string) bool {

	//Abrimos/creamos un archivo.
	file, err := os.OpenFile(path, os.O_RDWR, 0644)
	defer file.Close()
	if err != nil { //validar que no sea nulo.
		log.Fatal(err)
	}

	//Declaramos variable de tipo mbr
	m := mbr{}

	//Obtenemos el tamanio del mbr
	var size int = int(unsafe.Sizeof(m))

	//Lee la cantidad de <size> bytes del archivo
	data := leerBytesFdisk(file, size)

	//Convierte la data en un buffer,necesario para decodificar binario
	buffer := bytes.NewBuffer(data)

	//Decodificamos y guardamos en la variable m
	err = binary.Read(buffer, binary.BigEndian, &m)
	if err != nil {
		log.Fatal("binary.Read failed", err)
	}

	//obtengo el arreglo de particiones
	misParticiones := m.Particiones

	//obtengo el indice donde se encuentra la particion extendida
	posicionExtendida := 0
	for i := 0; i < 4; i++ {
		actual := misParticiones[i]
		if strings.Compare(strings.ToLower(string(actual.TipoParticion)), "e") == 0 {
			posicionExtendida = i
			break
		}
	}

	//accedo a particiones logicas
	misParticionesLogicas := misParticiones[posicionExtendida].ParticionesLogicas

	existe := false

	//recorro para ver si existe nombre
	for i := 0; i < 5; i++ {
		actual := misParticionesLogicas[i]

		//eliminando espacios en blanco o nulos del name
		nombrePart := ""
		for j := 0; j < 16; j++ {
			if actual.Name[j] != 0 {
				nombrePart += string(actual.Name[j])
			}
		}
		if strings.Compare(strings.ToLower(nombrePart), strings.ToLower(name)) == 0 {
			existe = true
			break
		}
	}

	//si existe logica con ese nombre
	if existe {
		return true //retorna que hay espacio
	}

	return false
}

//elimina particion logica
func eliminarParticionLogica(path string, name string) {
	//Abrimos/creamos un archivo.
	file, err := os.OpenFile(path, os.O_RDWR, 0644)
	defer file.Close()
	if err != nil { //validar que no sea nulo.
		log.Fatal(err)
	}

	//Declaramos variable de tipo mbr
	m := mbr{}

	//Obtenemos el tamanio del mbr
	var size int = int(unsafe.Sizeof(m))

	//Lee la cantidad de <size> bytes del archivo
	data := leerBytesFdisk(file, size)

	//Convierte la data en un buffer,necesario para decodificar binario
	buffer := bytes.NewBuffer(data)

	//Decodificamos y guardamos en la variable m
	err = binary.Read(buffer, binary.BigEndian, &m)
	if err != nil {
		log.Fatal("binary.Read failed", err)
	}

	//obtengo el arreglo de particiones
	misParticiones := m.Particiones

	//obtengo el indice donde se encuentra la particion extendida
	posicionExtendida := 0
	for i := 0; i < 4; i++ {
		actual := misParticiones[i]
		if strings.Compare(strings.ToLower(string(actual.TipoParticion)), "e") == 0 {
			posicionExtendida = i
			break
		}
	}

	//accedo a particiones logicas
	misParticionesLogicas := misParticiones[posicionExtendida].ParticionesLogicas

	posicionLogica := 0

	//recorro para ver si existe nombre
	for i := 0; i < 5; i++ {
		actual := misParticionesLogicas[i]

		//eliminando espacios en blanco o nulos del name
		nombrePart := ""
		for j := 0; j < 16; j++ {
			if actual.Name[j] != 0 {
				nombrePart += string(actual.Name[j])
			}
		}
		if strings.Compare(strings.ToLower(nombrePart), strings.ToLower(name)) == 0 {
			posicionLogica = i
			break
		}
	}

	//creo particion logica vacia
	particionLogicaNew := NodoParticionLogica{}

	//redireccionando next de particiones logicas
	//si la que se va eliminar esta en la primera posicion
	if posicionLogica == 0 {
		//solo se elimina
		misParticionesLogicas[posicionLogica] = particionLogicaNew

		//si la particiona eliminar esta en la ultima posicion
	} else if posicionLogica == 4 {
		//buscando particion anterior
		posAnterior := -1
		for i := 3; i > -1; i-- {
			//si encuantra antes una particion, guardo posicion donde la encuentra
			if misParticionesLogicas[i].Tamanio != 0 {
				posAnterior = i
				break
			}
		}
		//quiere decir que si encontro una particion antes
		if posAnterior != -1 {
			//la particion que esta antes ahora apunta a -1
			misParticionesLogicas[posAnterior].Next = int64(-1)
			misParticionesLogicas[posicionLogica] = particionLogicaNew

			//quiere decir que solo existe una particion, y es en la ultima
		} else {
			//solo se elimina
			misParticionesLogicas[posicionLogica] = particionLogicaNew
		}
		//si la particiona eliminar no es la primera ni la ultima
	} else {
		//buscando particion anterior
		posAnterior := -1
		for i := posicionLogica - 1; i > -1; i-- { //empieza a buscar una antes de la que se va eliminar
			//si encuantra antes una particion, guardo posicion donde la encuentra
			if misParticionesLogicas[i].Tamanio != 0 {
				posAnterior = i
				break
			}
		}

		//buscando particion siguiente
		posSiguiente := -1
		for i := posicionLogica + 1; i < 5; i++ { //empieza a buscar una despues de la que se va eliminar
			//si encuantra despues una particion, guardo posicion donde la encuentra
			if misParticionesLogicas[i].Tamanio != 0 {
				posSiguiente = i
				break
			}
		}

		//CASO 1: que tenga siguiente pero no anterior
		if (posSiguiente != -1) && (posAnterior == -1) {
			//solo se elimna
			misParticionesLogicas[posicionLogica] = particionLogicaNew

			//CASO 2: que tenga anterior pero no siguiente
		} else if (posSiguiente == -1) && (posAnterior != -1) {
			//next de parte anterior igual a -1
			misParticionesLogicas[posAnterior].Next = -1
			//elimino
			misParticionesLogicas[posicionLogica] = particionLogicaNew

			//CASO 3: que no tenga siguiente ni anterior
		} else if (posSiguiente == -1) && (posAnterior == -1) {
			//solo se elimna
			misParticionesLogicas[posicionLogica] = particionLogicaNew

			//CASO 4: que tenga siguiente y anterior
		} else if (posSiguiente != -1) && (posAnterior != -1) {
			//next de parte interior=next de parte a eliminar
			misParticionesLogicas[posAnterior].Next = misParticionesLogicas[posicionLogica].Next
			//elimino
			misParticionesLogicas[posicionLogica] = particionLogicaNew
		}

	}

	fmt.Println("\nDELETE LOGICA:")
	fmt.Println("	arr pos 0 Tamanio : ", misParticionesLogicas[0].Tamanio, " Star: ", misParticionesLogicas[0].Start, " Next: ", misParticionesLogicas[0].Next, " Tipo: ", string(misParticionesLogicas[0].TipoParticion))
	fmt.Println("	arr pos 1 Tamanio : ", misParticionesLogicas[1].Tamanio, " Star: ", misParticionesLogicas[1].Start, " Next: ", misParticionesLogicas[1].Next, " Tipo: ", string(misParticionesLogicas[1].TipoParticion))
	fmt.Println("	arr pos 2 Tamanio : ", misParticionesLogicas[2].Tamanio, " Star: ", misParticionesLogicas[2].Start, " Next: ", misParticionesLogicas[2].Next, " Tipo: ", string(misParticionesLogicas[2].TipoParticion))
	fmt.Println("	arr pos 3 Tamanio : ", misParticionesLogicas[3].Tamanio, " Star: ", misParticionesLogicas[3].Start, " Next: ", misParticionesLogicas[3].Next, " Tipo: ", string(misParticionesLogicas[3].TipoParticion))
	fmt.Println("	arr pos 4 Tamanio : ", misParticionesLogicas[4].Tamanio, " Star: ", misParticionesLogicas[4].Start, " Next: ", misParticionesLogicas[4].Next, " Tipo: ", string(misParticionesLogicas[4].TipoParticion))

	//las particiones logicas actuales se encuentran en 'misParticiones[posicionExtendida].ParticionesLogicas'
	//cuando se elimina una particion logica se elimina de 'misParticionesPrimarias'
	//entonces 'misParticionesPrimarias' tienen las actuales, y se quito la que se elimino
	//por eso se iguala de nuevo, para que 'misParticiones[posicionExtendida].ParticionesLogicas', se actualice con la particion eliminada
	misParticiones[posicionExtendida].ParticionesLogicas = misParticionesLogicas

	//para que se actualice nada mas
	m.Particiones = misParticiones

	file.Seek(0, 0)
	s1 := &m

	//Reescribimos struct (MBR)
	var binario3 bytes.Buffer
	binary.Write(&binario3, binary.BigEndian, s1)
	escribirBytes(file, binario3.Bytes())

}

//valida si se le puede agregar o quitar espacio en primaria o extendida
func validaSiSeLePuedeaddEspacioEnPrimariaExtendida(path string, name string, unit string, add int64) bool {

	//se hace la convercion de kb a bytes, o mb a bytes, segun sea el caso
	if strings.Compare(strings.ToLower(unit), "k") == 0 {
		add = add * 1024
	} else if strings.Compare(strings.ToLower(unit), "m") == 0 {
		add = add * 1024 * 1024
	}

	//Abrimos/creamos un archivo.
	file, err := os.OpenFile(path, os.O_RDWR, 0644)
	defer file.Close()
	if err != nil { //validar que no sea nulo.
		log.Fatal(err)
	}

	//Declaramos variable de tipo mbr
	m := mbr{}

	//Obtenemos el tamanio del mbr
	var size int = int(unsafe.Sizeof(m))

	//Lee la cantidad de <size> bytes del archivo
	data := leerBytesFdisk(file, size)

	//Convierte la data en un buffer,necesario para decodificar binario
	buffer := bytes.NewBuffer(data)

	//Decodificamos y guardamos en la variable m
	err = binary.Read(buffer, binary.BigEndian, &m)
	if err != nil {
		log.Fatal("binary.Read failed", err)
	}

	//obtengo el arreglo de particiones
	misParticiones := m.Particiones

	posicionParticion := 0

	//recorro para ver en que posicion esta la particion con el name
	for i := 0; i < 4; i++ {
		actual := misParticiones[i]

		//eliminando espacios en blanco o nulos del name
		nombrePart := ""
		for j := 0; j < 16; j++ {
			if actual.Name[j] != 0 {
				nombrePart += string(actual.Name[j])
			}
		}
		if strings.Compare(strings.ToLower(nombrePart), strings.ToLower(name)) == 0 {
			posicionParticion = i
			break
		}
	}

	//si hay que agregar en la primera posicion
	if posicionParticion == 0 {

		//se agregara espacio
		if add >= 0 {

			//buscando particion siguiente
			posSiguiente := -1
			for i := posicionParticion + 1; i < 4; i++ { //empieza a buscar una despues en la que se va agregar
				//si encuantra despues una particion, guardo posicion donde la encuentra
				if misParticiones[i].Tamanio != 0 {
					posSiguiente = i
					break
				}
			}

			//si hay una particion despues
			if posSiguiente != -1 {
				//obtengo el star de la particion a la que le quiero agregar
				starActual := misParticiones[posicionParticion].Start
				//obtengo el tamanio de la particon a la que le quiero agregar
				tamanioParticion := misParticiones[posicionParticion].Tamanio
				//obtengo el star de la posicion siguiente
				starSiguiente := misParticiones[posSiguiente].Start
				//opero para que me de el espacio que hay entre las dos
				espacioDisponible := starSiguiente - (starActual + tamanioParticion)
				//valido
				if espacioDisponible >= add {
					//si se puede agregar espacio
					return true
					//newTamanaio := tamanioParticion + add
					//misParticiones[posicionParticion].Tamanio = newTamanaio
				}

				//si no hay particion despues
			} else if posSiguiente == -1 {
				//obtengo el tamanio del disco
				tamDisco := m.Tamanio
				//obtengo el star de la particion a la que le quiero agregar
				starActual := misParticiones[posicionParticion].Start
				//obtengo el tamanio de la particon a la que le quiero agregar
				tamanioParticion := misParticiones[posicionParticion].Tamanio
				//opero para que me del el espacio disponible
				espacioDisponible := tamDisco - (starActual + tamanioParticion)
				//valido
				if espacioDisponible >= add {
					//se puede agregar espacio
					return true
				}
			}

			//se quitara espacio
		} else if add < 0 {
			//obtenemos tamanio de particion que le queremos quitar espacio
			tamanioParticion := misParticiones[posicionParticion].Tamanio
			//es suma porque el size viene negativo
			nuevoTamanio := tamanioParticion + add
			//validamos
			if nuevoTamanio >= 1 {
				return true
			}
		}

		//si hay que agregar en la ultima posicion
	} else if posicionParticion == 3 {

		//si hay que agregar espacio
		if add >= 0 {
			//obtengo el tamanio del disco
			tamDisco := m.Tamanio
			//obtengo el star de la particion a la que le quiero agregar
			starActual := misParticiones[posicionParticion].Start
			//obtengo el tamanio de la particon a la que le quiero agregar
			tamanioParticion := misParticiones[posicionParticion].Tamanio
			//opero para que me del el espacio disponible
			espacioDisponible := tamDisco - (starActual + tamanioParticion)
			//valido
			if espacioDisponible >= add {
				//se puede agregar espacio
				return true
			}

			//si hay que quitar espacio
		} else if add < 0 {
			//obtenemos tamanio de particion que le queremos quitar espacio
			tamanioParticion := misParticiones[posicionParticion].Tamanio
			//es suma porque el size viene negativo
			nuevoTamanio := tamanioParticion + add
			//validamos
			if nuevoTamanio >= 1 {
				return true
			}
		}

		//no es la primera ni la ultima posicion (puede ser dos o tres)
	} else {
		if add >= 0 {

			//buscando particion siguiente
			posSiguiente := -1
			for i := posicionParticion + 1; i < 4; i++ { //empieza a buscar una despues en la que se va agregar
				//si encuantra despues una particion, guardo posicion donde la encuentra
				if misParticiones[i].Tamanio != 0 {
					posSiguiente = i
					break
				}
			}

			//tenga siguiente
			if posSiguiente != -1 {
				//star siguiente
				starSiguiente := misParticiones[posSiguiente].Start
				//star acutal
				starActual := misParticiones[posicionParticion].Start
				//tamanio actual
				tamanioParticion := misParticiones[posicionParticion].Tamanio
				//espacio libre
				espacioDisponible := starSiguiente - (starActual + tamanioParticion)
				//valido
				if espacioDisponible >= add {
					return true
				}

				//no tenga siguiente
			} else if posSiguiente == -1 {
				//obtengo el tamanio del disco
				tamDisco := m.Tamanio
				//obtengo el star de la particion a la que le quiero agregar
				starActual := misParticiones[posicionParticion].Start
				//obtengo el tamanio de la particon a la que le quiero agregar
				tamanioParticion := misParticiones[posicionParticion].Tamanio
				//opero para que me del el espacio disponible
				espacioDisponible := tamDisco - (starActual + tamanioParticion)
				//valido
				if espacioDisponible >= add {
					//se puede agregar espacio
					return true
				}
			}

		} else if add < 0 {
			//obtenemos tamanio de particion que le queremos quitar espacio
			tamanioParticion := misParticiones[posicionParticion].Tamanio
			//es suma porque el size viene negativo
			nuevoTamanio := tamanioParticion + add
			//validamos
			if nuevoTamanio >= 1 {
				return true
			}
		}
	}

	return false
}

//agrega o quita espacio en primaria o extendida
func addEspacioEnPrimariaExtendida(path string, name string, unit string, add int64) {

	//se hace la convercion de kb a bytes, o mb a bytes, segun sea el caso
	if strings.Compare(strings.ToLower(unit), "k") == 0 {
		add = add * 1024
	} else if strings.Compare(strings.ToLower(unit), "m") == 0 {
		add = add * 1024 * 1024
	}

	//Abrimos/creamos un archivo.
	file, err := os.OpenFile(path, os.O_RDWR, 0644)
	defer file.Close()
	if err != nil { //validar que no sea nulo.
		log.Fatal(err)
	}

	//Declaramos variable de tipo mbr
	m := mbr{}

	//Obtenemos el tamanio del mbr
	var size int = int(unsafe.Sizeof(m))

	//Lee la cantidad de <size> bytes del archivo
	data := leerBytesFdisk(file, size)

	//Convierte la data en un buffer,necesario para decodificar binario
	buffer := bytes.NewBuffer(data)

	//Decodificamos y guardamos en la variable m
	err = binary.Read(buffer, binary.BigEndian, &m)
	if err != nil {
		log.Fatal("binary.Read failed", err)
	}

	//obtengo el arreglo de particiones
	misParticiones := m.Particiones

	posicionParticion := 0

	//recorro para ver en que posicion esta la particion con el name
	for i := 0; i < 4; i++ {
		actual := misParticiones[i]

		//eliminando espacios en blanco o nulos del name
		nombrePart := ""
		for j := 0; j < 16; j++ {
			if actual.Name[j] != 0 {
				nombrePart += string(actual.Name[j])
			}
		}
		if strings.Compare(strings.ToLower(nombrePart), strings.ToLower(name)) == 0 {
			posicionParticion = i
			break
		}
	}

	//si hay que agregar en la primera posicion
	if posicionParticion == 0 {

		//se agregara espacio
		if add >= 0 {

			//buscando particion siguiente
			posSiguiente := -1
			for i := posicionParticion + 1; i < 4; i++ { //empieza a buscar una despues en la que se va agregar
				//si encuantra despues una particion, guardo posicion donde la encuentra
				if misParticiones[i].Tamanio != 0 {
					posSiguiente = i
					break
				}
			}

			//si hay una particion despues
			if posSiguiente != -1 {
				//obtengo el star de la particion a la que le quiero agregar
				starActual := misParticiones[posicionParticion].Start
				//obtengo el tamanio de la particon a la que le quiero agregar
				tamanioParticion := misParticiones[posicionParticion].Tamanio
				//obtengo el star de la posicion siguiente
				starSiguiente := misParticiones[posSiguiente].Start
				//opero para que me de el espacio que hay entre las dos
				espacioDisponible := starSiguiente - (starActual + tamanioParticion)
				//valido
				if espacioDisponible >= add {
					//si se puede agregar espacio
					newTamanaio := tamanioParticion + add
					misParticiones[posicionParticion].Tamanio = newTamanaio
				}

				//si no hay particion despues
			} else if posSiguiente == -1 {
				//obtengo el tamanio del disco
				tamDisco := m.Tamanio
				//obtengo el star de la particion a la que le quiero agregar
				starActual := misParticiones[posicionParticion].Start
				//obtengo el tamanio de la particon a la que le quiero agregar
				tamanioParticion := misParticiones[posicionParticion].Tamanio
				//opero para que me del el espacio disponible
				espacioDisponible := tamDisco - (starActual + tamanioParticion)
				//valido
				if espacioDisponible >= add {
					//se puede agregar espacio
					newTamanaio := tamanioParticion + add
					misParticiones[posicionParticion].Tamanio = newTamanaio
				}
			}

			//se quitara espacio
		} else if add < 0 {
			//obtenemos tamanio de particion que le queremos quitar espacio
			tamanioParticion := misParticiones[posicionParticion].Tamanio
			//es suma porque el size viene negativo
			nuevoTamanio := tamanioParticion + add
			//validamos
			if nuevoTamanio >= 1 {
				misParticiones[posicionParticion].Tamanio = nuevoTamanio
			}
		}

		//si hay que agregar en la ultima posicion
	} else if posicionParticion == 3 {

		//si hay que agregar espacio
		if add >= 0 {
			//obtengo el tamanio del disco
			tamDisco := m.Tamanio
			//obtengo el star de la particion a la que le quiero agregar
			starActual := misParticiones[posicionParticion].Start
			//obtengo el tamanio de la particon a la que le quiero agregar
			tamanioParticion := misParticiones[posicionParticion].Tamanio
			//opero para que me del el espacio disponible
			espacioDisponible := tamDisco - (starActual + tamanioParticion)
			//valido
			if espacioDisponible >= add {
				//se puede agregar espacio
				newTamanaio := tamanioParticion + add
				misParticiones[posicionParticion].Tamanio = newTamanaio
			}

			//si hay que quitar espacio
		} else if add < 0 {
			//obtenemos tamanio de particion que le queremos quitar espacio
			tamanioParticion := misParticiones[posicionParticion].Tamanio
			//es suma porque el size viene negativo
			nuevoTamanio := tamanioParticion + add
			//validamos
			if nuevoTamanio >= 1 {
				misParticiones[posicionParticion].Tamanio = nuevoTamanio
			}
		}

		//no es la primera ni la ultima posicion (puede ser dos o tres)
	} else {
		if add >= 0 {

			//buscando particion siguiente
			posSiguiente := -1
			for i := posicionParticion + 1; i < 4; i++ { //empieza a buscar una despues en la que se va agregar
				//si encuantra despues una particion, guardo posicion donde la encuentra
				if misParticiones[i].Tamanio != 0 {
					posSiguiente = i
					break
				}
			}

			//tenga siguiente
			if posSiguiente != -1 {
				//star siguiente
				starSiguiente := misParticiones[posSiguiente].Start
				//star acutal
				starActual := misParticiones[posicionParticion].Start
				//tamanio actual
				tamanioParticion := misParticiones[posicionParticion].Tamanio
				//espacio libre
				espacioDisponible := starSiguiente - (starActual + tamanioParticion)
				//valido
				if espacioDisponible >= add {
					//se puede agregar espacio
					newTamanaio := tamanioParticion + add
					misParticiones[posicionParticion].Tamanio = newTamanaio
				}

				//no tenga siguiente
			} else if posSiguiente == -1 {
				//obtengo el tamanio del disco
				tamDisco := m.Tamanio
				//obtengo el star de la particion a la que le quiero agregar
				starActual := misParticiones[posicionParticion].Start
				//obtengo el tamanio de la particon a la que le quiero agregar
				tamanioParticion := misParticiones[posicionParticion].Tamanio
				//opero para que me del el espacio disponible
				espacioDisponible := tamDisco - (starActual + tamanioParticion)
				//valido
				if espacioDisponible >= add {
					//se puede agregar espacio
					newTamanaio := tamanioParticion + add
					misParticiones[posicionParticion].Tamanio = newTamanaio
				}
			}

		} else if add < 0 {
			//obtenemos tamanio de particion que le queremos quitar espacio
			tamanioParticion := misParticiones[posicionParticion].Tamanio
			//es suma porque el size viene negativo
			nuevoTamanio := tamanioParticion + add
			//validamos
			if nuevoTamanio >= 1 {
				misParticiones[posicionParticion].Tamanio = nuevoTamanio
			}
		}
	}

	fmt.Println("\nSE ADD O DELETE ESPACIO EN PRIMARIA O EXTENDIDA:")
	fmt.Println("	arr pos 0 Tamanio : ", misParticiones[0].Tamanio, " Star: ", misParticiones[0].Start, " Tipo: ", string(misParticiones[0].TipoParticion))
	fmt.Println("	arr pos 1 Tamanio : ", misParticiones[1].Tamanio, " Star: ", misParticiones[1].Start, " Tipo: ", string(misParticiones[1].TipoParticion))
	fmt.Println("	arr pos 2 Tamanio : ", misParticiones[2].Tamanio, " Star: ", misParticiones[2].Start, " Tipo: ", string(misParticiones[2].TipoParticion))
	fmt.Println("	arr pos 3 Tamanio : ", misParticiones[3].Tamanio, " Star: ", misParticiones[3].Start, " Tipo: ", string(misParticiones[3].TipoParticion))

	//las particiones actuales en el disco se encuentran en 'm.particiones'
	//cuando se modifica el tamanio de una nueva particion se modifica en 'misPartiiones'
	//entonces 'misParticiones' tiene las anteriores, y la que se modifico ahora
	//por eso se iguala de nuevo, para que em 'm.Particiones' se guarden particiones ya actualizadas
	m.Particiones = misParticiones

	file.Seek(0, 0)
	s1 := &m

	//Reescribimos struct (MBR)
	var binario3 bytes.Buffer
	binary.Write(&binario3, binary.BigEndian, s1)
	escribirBytes(file, binario3.Bytes())

}

//Función que lee del archivo, se especifica cuantos bytes se quieren leer.
func leerBytesFdisk(file *os.File, number int) []byte {
	bytes := make([]byte, number) //array de bytes, de tamanio que recibe

	_, err := file.Read(bytes) // Leido -> bytes
	if err != nil {
		log.Fatal(err)
	}

	return bytes
}

//-------------------------------FIN FDISK-------------------------------//

//-------------------------------INICIO MOUNT-------------------------------//
func mountComando(index int) {
	path := ""
	name := ""

	for i := index; i < len(listaComandos); i++ {

		if strings.Compare(strings.ToLower(listaComandos[i]), "path") == 0 { //cuando encuentre palabra reservada path
			if (strings.Compare(listaComandos[i-1], "-") == 0) && (strings.Compare(listaComandos[i+1], "->") == 0) { // validar si esta de esta forma -path->
				ruta := listaComandos[i+2]        //ruta
				if strings.Contains(ruta, "\"") { //si la ruta que viene contiene comillas
					ruta2 := ruta[1 : len(ruta)-1] //le quitamos comillas a la ruta
					path = ruta2                   //funcion que leera el archivo
				} else { //sino tiene comillas manda la ruta normal
					path = ruta
				}
			} else {
				fmt.Print("\n[ ERROR: comando 'MOUNT' -> 'path' ]")
			}
		} else if strings.Compare(strings.ToLower(listaComandos[i]), "name") == 0 {
			if (strings.Compare(listaComandos[i-1], "-") == 0) && (strings.Compare(listaComandos[i+1], "->") == 0) { // validar si esta de esta forma -name->
				name = listaComandos[i+2] //name
			} else {
				fmt.Print("\n[ ERROR: comando 'MOUNT' -> 'name' ]")
			}
		}
	}

	montarParticion(path, name)
}

func montarParticion(path string, name string) {

}

//-------------------------------FIN MOUNT-------------------------------//

//-------------------------------INICIO UNMOUNT-------------------------------//
func unmountComando(index int) {
	idn := ""

	for i := index; i < len(listaComandos); i++ {

		if strings.Compare(strings.ToLower(listaComandos[i]), "idn") == 0 {
			if (strings.Compare(listaComandos[i-1], "-") == 0) && (strings.Compare(listaComandos[i+1], "->") == 0) { // validar si esta de esta forma -name->
				idn = listaComandos[i+2] //name
			} else {
				fmt.Print("\n[ ERROR: comando 'MOUNT' -> 'name' ]")
			}
		}
	}

	desmontarParticion(idn)
}

func desmontarParticion(path string) {

}

//-------------------------------FIN MOUNT-------------------------------//

//-------------------------------OPERACIONES PARA MBR-------------------------------//

//---------------------------------REPORTE MBR---------------------------------//

func reporteMBR(path string) {
	//Abrimos/creamos un archivo.
	file, err := os.OpenFile(path, os.O_RDWR, 0644)
	defer file.Close()
	if err != nil { //validar que no sea nulo.
		log.Fatal(err)
	}

	//Declaramos variable de tipo mbr
	m := mbr{}

	//Obtenemos el tamanio del mbr
	var size int = int(unsafe.Sizeof(m))

	//Lee la cantidad de <size> bytes del archivo
	data := leerBytesFdisk(file, size)

	//Convierte la data en un buffer,necesario para decodificar binario
	buffer := bytes.NewBuffer(data)

	//Decodificamos y guardamos en la variable m
	err = binary.Read(buffer, binary.BigEndian, &m)
	if err != nil {
		log.Fatal("binary.Read failed", err)
	}

	//obtengo el arreglo de particiones
	misParticiones := m.Particiones

	cadenaRep := "digraph { \n\n"
	cadenaRep += "tbl [ \n\n"
	cadenaRep += "shape=plaintext \n"
	cadenaRep += "label=< \n\n"
	cadenaRep += "<table color='orange' cellspacing='0'>\n"

	//cabecera de tabla
	cadenaRep += "<tr><td>NOMBRE</td><td>VALOR</td></tr>\n"

	//datos MBR
	fechaMbr := ""
	//si hay espacios en blanco en el [20]byte, osea espacios nulos, tira error
	for i := 0; i < 20; i++ {
		if m.Fecha[i] != 0 { //los que sean nulos no los concatena
			fechaMbr += string(m.Fecha[i])
		}
	}

	cadenaRep += "<tr><td>MBR_Tamanio</td><td>" + strconv.Itoa(int(m.Tamanio)) + "</td></tr>\n"
	cadenaRep += "<tr><td>MBR_Fecha</td><td>" + fechaMbr + "</td></tr>\n"
	cadenaRep += "<tr><td>MBR_Asignacion</td><td>" + strconv.Itoa(int(m.NumAsignacion)) + "</td></tr>\n"

	//r
	for i := 0; i < 4; i++ {
		actual := misParticiones[i]

		if actual.Tamanio != 0 {

			//eliminando espacios nulos
			nombrePart := ""
			for i := 0; i < 16; i++ {
				if actual.Name[i] != 0 { //los que sean nulos no los concatena
					nombrePart += string(actual.Name[i])
				}
			}

			cadenaRep += "<tr><td>" + nombrePart + "_Name</td><td>" + nombrePart + "</td></tr>\n"
			cadenaRep += "<tr><td>" + nombrePart + "_Size</td><td>" + strconv.Itoa(int(actual.Tamanio)) + "</td></tr>\n"
			cadenaRep += "<tr><td>" + nombrePart + "_Tipe</td><td>" + string(actual.TipoParticion) + "</td></tr>\n"
			cadenaRep += "<tr><td>" + nombrePart + "_Fit</td><td>" + string(actual.TipoAjuste[:]) + "</td></tr>\n"
			cadenaRep += "<tr><td>" + nombrePart + "_Start</td><td>" + strconv.Itoa(int(actual.Start)) + "</td></tr>\n"
			cadenaRep += "<tr><td>" + nombrePart + "_Status</td><td>" + strconv.Itoa(int(actual.Estado)) + "</td></tr>\n"
		}
	}

	cadenaRep += "</table>\n\n"
	cadenaRep += ">];\n"
	cadenaRep += "}"

	crearDot("report_mbr", cadenaRep)
	crearImg("report_mbr")
}

func crearDot(name string, cadena string) {
	f := createFile(name + ".dot")
	defer closeFile(f)
	writeFile(f, cadena)
}

func createFile(p string) *os.File {
	fmt.Println("crear")
	f, err := os.Create(p)
	if err != nil {
		panic(err)
	}
	return f
}

func writeFile(f *os.File, cadena string) {
	fmt.Println("escribir")
	fmt.Fprintln(f, cadena)

}

func closeFile(f *os.File) {
	fmt.Println("cerrar")
	f.Close()
}

func crearImg(name string) {
	//ejecutando comando, en tiempo de ejecucion
	cmd := exec.Command("dot", "-Tpng", name+".dot", "-o", name+".png")
	var waitStatus syscall.WaitStatus
	if err := cmd.Run(); err != nil {
		if err != nil {
			os.Stderr.WriteString(fmt.Sprintf("Error: %s\n", err.Error()))
		}
		if exitError, ok := err.(*exec.ExitError); ok {
			waitStatus = exitError.Sys().(syscall.WaitStatus)
			fmt.Printf("Output: %s\n", []byte(fmt.Sprintf("%d", waitStatus.ExitStatus())))
		}
	} else {
		// Success
		waitStatus = cmd.ProcessState.Sys().(syscall.WaitStatus)
		//fmt.Printf("Output: %s\n", []byte(fmt.Sprintf("%d", waitStatus.ExitStatus())))
	}
}

//-------------------------------FIN REPORTE MBR-------------------------------//
