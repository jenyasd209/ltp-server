# Ltp app

## Details

This application fetches the last traded prices from the Kraken API for the following pairs:

- BTC/USD 
- ETH/USD  
- LTC/USD

To get the prices use the next route `/api/v1/ltp` (example [localhost:8080/api/v1/ltp](localhost:8080/api/v1/ltp)).

## Requirements

- Ensure Docker is installed and running on your machine.
- The 8080 port is free (`-p 8080:8080`) to use it for the container. Or use another one (`-p 8081:8080`).

## Build and Run the Application

1. **Clone the repository** (if you haven't already):

```sh
    git clone https://github.com/jenyasd209/ltp-server
    cd ltp-server
```

2. **Build the Docker image**:

```sh
    docker build -t ltp-server .
```

Note: docker builder prune --filter ancestor=ltp

3. **Run the Docker container**:

```sh
    docker run -d --name ltp -p 8080:8080 ltp-server
```

4. **Check the logs**:

```sh
    docker logs -f ltp
```

## Stopping the Application

1. Stop the container

```sh
docker stop ltp
```

2. Delete container and image 
```sh
docker rm ltp && docker rmi ltp-server
```