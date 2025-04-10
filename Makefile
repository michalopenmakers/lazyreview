.PHONY: build run clean macapp

BINARY=lazyreview
APP_NAME=LazyReview

all: clean build macapp

build:
	go build -o $(BINARY) .

run: build
	./$(BINARY)

clean:
	rm -f $(BINARY)
	rm -rf $(APP_NAME).app

macapp:
	fyne package -os darwin -name $(APP_NAME) -icon icon.png