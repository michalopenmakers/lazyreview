.PHONY: build run clean


BINARY=lazyreview

all: clean build run

build:
	@echo "Kompilacja aplikacji..."
	go build -o $(BINARY) .

run: build
	@echo "Uruchamianie aplikacji..."
	./$(BINARY)

clean:
	@echo "Czyszczenie builda..."
	rm -f $(BINARY)
