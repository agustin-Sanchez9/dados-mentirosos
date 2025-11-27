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
│       └── main.go       # Punto de entrada. Solo inicializa y arranca el server.
├── internal/             # Código privado de tu aplicación (no importable por otros)
│   ├── game/             # LÓGICA PURA DEL JUEGO (Tu "Core")
│   │   ├── engine.go     # La máquina de estados del juego (turnos, reglas).
│   │   ├── player.go     # Structs del jugador (dados, nombre, conexión).
│   │   └── room.go       # Lógica de la sala y broadcast de mensajes.
│   ├── handlers/         # CAPA HTTP & WEBSOCKETS
│   │   ├── http.go       # Handlers para servir el HTML inicial y HTMX.
│   │   └── ws.go         # Manejo de conexiones WebSocket (Upgrader).
│   └── templates/        # LÓGICA DE RENDERIZADO
│       └── renderer.go   # Funciones helpers para parsear y ejecutar templates.
├── ui/                   # FRONTEND (Archivos estáticos y plantillas)
│   ├── static/           # Archivos públicos
│   │   ├── css/          # Tu archivo de Tailwind (output.css).
│   │   └── js/           # htmx.min.js (o descárgalo de CDN).
│   └── html/             # TUS PLANTILLAS HTML
│       ├── base.html     # El esqueleto (<html><head>...).
│       ├── home.html     # La pantalla de "Crear Sala".
│       ├── game.html     # La pantalla principal del juego.
│       └── partials/     # FRAGMENTOS PARA HTMX (Lo más importante)
│           ├── board.html    # El tablero central (se recarga mucho).
│           ├── dice.html     # Cómo se ve un dado.
│           └── controls.html # Botones (Apostar/Mentiroso).
├── go.mod
├── go.sum
└── Makefile              # Comandos para correr el server y compilar CSS.
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
