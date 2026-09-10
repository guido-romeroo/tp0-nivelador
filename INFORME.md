**Alumno Guido Nadal, Romero Krause. 111877**
# Informe
## Descripción
En el presente trabajo se implmentó un *sistema de Lotería*, en donde las **Agencias** envían las apuestas que tienen cada una de ellas a la **Lotería Nacional**.  
La Lotería Nacional realiza el sorteo e informa a las agencias las apuestas ganadoras que estaban presentes en el conjunto que cada una informó.

## Protocolo de Comunicación
### A nivel de Aplicación
Cada mensaje posee un tipo y según el caso un payload.  
Visto en bytes: 
```
+------+------------------------+
| tipo |        payload         |
+------+------------------------+
  1B          variable
```

Tipos de mensaje (`protocol.py` / `protocol.go`):
 
| Tipo    | Valor  | Dirección       | Payload                           |
|---------|--------|-----------------|-----------------------------------|
| `BETS`  | `0x01` | cliente→server  | una o más apuestas                |
| `WINNER`| `0x02` | server→cliente  | una apuesta ganadora              |
| `ACK`   | `0x03` | server→cliente  | vacío                             |
| `NACK`  | `0x04` | server→cliente  | vacío                             |
| `BYE`   | `0x05` | ambos sentidos  | vacío                             |

#### Serialización
Cada Bet (que viaja tanto en mensajes de tipo `BETS` y `WINNER`) **se serializa con un esquema mixto**: *Longitud implícita* para los campos numéricos y *explícita* para los de tipo texto.

```
agency_id(2B) | len(first_name)(1B) | first_name | len(last_name)(1B) | last_name
| document(4B) | len(birthdate)(1B) | birthdate | number(4B)
```
Como cada campo se autodelimita, al enviar batchs de Bets no es necesario agregar delimitadores extras.

### A nivel de Red
Me refiero a red a cómo se utiliza el safe_socket, ya que debe explicitarse cuánto se desea recibir en el recv, y esto como tal es independiente del protocolo definido por nuestro sistema.  
Simplemente se abstrae esta complejidad en el tipo `ConnectionFacilitator` el cual utiliza un header de tamaño fijo para explicitar el largo de los mensajes, seguido del mensaje serializado.

## Concurrencia y sincronización
 
El servidor utiliza **pasaje de mensajes** para evitar la sincronización explícita:
- **1 thread acceptor** (`accept_connections`): escucha nuevas conexiones y lanza un hilo por cada una.
- **1 thread por conexión aceptada** (`_handle_client`): recibe apuestas, se comunica con un coordinador de sorteo para persistirlas y envía los ganadores. 
- **1 thread coordinador de sorteo** (`coordinate_raffle`): es el dueño de la instancia de `Lottery`. Como es el único lector/escritor, no necesita locks.
- **`coordinator_channel`** (`queue.Queue`): canal utilizado por todos los threads que atienden clientes. Le mandan pedidos al coordinador (`BETS`, `START_RAFFLE`, `FINISHED`).
- Cada pedido va acompañado de un **canal de respuesta de un solo uso** (`ack_channel` para confirmar el guardado, `winners_channel` para recibir la lista de ganadores). 

**Quorum:** el coordinador cuenta cuántas agencias mandaron `START_RAFFLE` en total. Se espera usa una sola vez, cuando se alcanza `AGENCY_QUORUM_MIN` por primera vez, se calculan los ganadores y se les responde a las agencias que estaban esperando. 
Cualquier agencia que llegue *después* de cumplirse el quorum recibe su resultado de inmediato.
 
**Apagado ordenado (SIGTERM):** un `threading.Event` marca el estado de apagado. El handler de la señal:
1. Setea el evento de `shutdown`
2. encola un mensaje `SHUTDOWN` en `coordinator_channel` para destrabar al coordinador en el caso de que se encuentre esperando por un mensaje del canal.
3. cierra el socket del escucha para destrabar el `accept()`.
4. cierra los sockets de clientes para destrabarlos de cualquier `recv`/`send`.

El método `graceful_shutdown` usado por el coordinator permite destrabar a hilos de clientes bloqueados en cualquier canal de respuesta (ya sea de ack o winners).

## ¿Por qué el GIL no es un problema para este sistema?
 
El *Global Interpreter Lock* impide que dos threads ejecuten bytecode de Python **al mismo tiempo**, pero **se libera automáticamente durante las llamadas bloqueantes de I/O** (`socket.recv`/`send`, lectura/escritura de archivos, get de queues). Como este servidor no realiza operaciones CPU-Bound, sino que cada thread pasa gran parte del tiempo bloqueado, esperando datos (de red o de canales `Queue`), el GIL estará constantemente liberado, permitiendo el avance.  
El cuello de botella de este sistema es la red y el disco, no la CPU.