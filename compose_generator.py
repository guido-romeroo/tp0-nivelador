import sys
def check_args():
    if len(sys.argv) != 2:
        print(f"Utilizá {sys.argv[0]} <cantidad_de_clientes> para ejecutar")
        return None

    try:
        cantidad = int(sys.argv[1])
    except ValueError:
        print("El argumento debe ser un entero")
        return None

    if cantidad < 1:
        print("La cantidad de clientes debe ser mayor a 0")
        return None

    return cantidad
    
def cliente_str(id):
    return f"""  client_{id}:
    build:
      context: ./services/client
      dockerfile: Dockerfile
    container_name: client_{id}
    depends_on:
      - server
    environment:
      - AGENCY_ID={id}
      - SERVER_HOST=server
      - SERVER_PORT=5678

"""

def reescribir_archivo(cantidad_clientes):
    inicio = """services:
  server:
    build:
      context: ./services/server
      dockerfile: Dockerfile
    container_name: server
    environment:
      - PYTHONUNBUFFERED=1
      - SERVER_HOST=server
      - SERVER_PORT=5678
      
"""
    with open("docker-compose.yaml", "w") as archivo:
        archivo.write(inicio)
        for id in range(cantidad_clientes):
            archivo.write(cliente_str(id))

def main():
    cantidad_clientes = check_args()
    if cantidad_clientes is None:
        return
    
    reescribir_archivo(cantidad_clientes)

if __name__ == "__main__":
    main()