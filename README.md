# dados-mentirosos
Repositorio principal para el desarrollo del juego web "Dados Mentirosos". Estrategia Backend-first con HTMX y GO. 

## Idea general del juego:
Cada jugador dispondra de un set de dados (todos misma cantidad entre 3 a 6). Al comenzar el juego cada jugador lanza sus dados para obtener una configuracion de numeros. El jugador solo podra ver los dados propios, los dados del resto de jugadores seran ocultos para el. Luego comienza la seccion de turnos, en la cual uno a uno los jugadores haran sus apuestas sobre la cantidad de dados que existen, en total contando los de todos los jugadores, de un cierto valor. Si un jugador considera que la apuesta de otro es poco probable puede llamarlo mentiroso, en cuyo caso los dados de todos los jugadores seran visibles, para verificar si la apuesta era o no una mentira. Para realizar una apuesta un jugador puede cambiar el valor del dado a contar, pero si o si debe apostar por un numero mayor que la ultima apuesta.

Ejemplo (el primer numero es la cantidad y el segundo el dado):
player1 apuesta a 4 - 6.
player2 apuesta a 5 - 3. 
player3 llama mentiroso a player2.

## Estructura del proyecto:
```text
dados-mentirosos/
├── cmd/
│   └── server/
│       └── main.go           # Arranca HTTP y Websockets
├── internal/
│   ├── game/                 
│   │   ├── lobby.go          # Crear sala, unir jugador, guardar configs.
|   |   ├── manager.go        # Gestiona las salas activas del servidor.
│   │   ├── round.go          # Lógica de apuestas, turnos, mentirosos.
│   │   └── types.go          # Structs (Room, Player, Config).
│   └── handlers/             # MANEJADORES DE RUTAS
│       ├── http.go           # GET /, POST /create, POST /enter
│       └── ws.go             # Websockets del juego.
└── ui/
   └── html/
       ├── base.html         # <html>, <head>, <body> container principal
       ├── pages/            
       │   ├── home.html     # Pantalla inicial con el formulario para unirse o crear sala.
       │   ├── lobby.html    # Pantalla de la sala, con lista de jugadores y configuraciones.
       │   └── game.html     # Pantalla de juego.
       └── partials/         # COMPONENTES REUTILIZABLES (Lo que HTMX actualiza)
           ├── game/
           │   ├── screen.html    # Pantalla base del juego que muestra los jugadores y la apuesta actual
           │   ├── results.html   # Pantalla que muestra los resultados
           │   └── controls.html  # ui de controles para apuestas y para llamar mentiroso
           └── lobby/
               └── settings.html  # ui de configuraciones para el usuario

```

## Cuestiones basicas
- El juego no requerira que las personas deban crear una cuenta ni iniciar sesion, tan solo se les pedira que ingresen un nombre para ser reconocido por los demas. Este nombre puede ser lo que las personas quieren.
- El jugador puede unirse a una sale mediante el codigo de la misma, el cual es provisto al creador para invitar a quien desee.
- El jugador podra crear una sala deeterminando sus configuraciones:
    - Cantidad de dados (3 a 6).
    - Cantidad de jugadores (2 a 7).
    - Duracion de los turnos en segundos (30, 60, 90 o inf).
    - Minimo de incremento de apuesta por ronda (1, 2, o 3).
    - Los 1 son comodines, es decir cuentan para la suma de todos los dados.
- Una vez finalizada la partida los jugadores podran empezar una nueva o volver al menu de inicio.

