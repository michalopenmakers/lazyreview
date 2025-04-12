# LazyReview

[![Go](https://github.com/michalopenmakers/lazyreview/actions/workflows/go.yml/badge.svg)](https://github.com/michalopenmakers/lazyreview/actions/workflows/go.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/michalopenmakers/lazyreview)](https://goreportcard.com/report/github.com/michalopenmakers/lazyreview)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT)

LazyReview is an AI-powered code review assistant that automatically analyzes pull requests and provides intelligent feedback.

## Features

- 🤖 Automated code reviews
- 🔍 Detects common issues and anti-patterns
- 💡 Suggests improvements and best practices
- 🚀 Seamless integration with GitHub

## Installation

### Prerequisites

- Go 1.18 or higher
- [Fyne](https://fyne.io/) dependencies for GUI

### From Source

```bash
# Clone the repository
git clone https://github.com/michalopenmakers/lazyreview.git
cd lazyreview

# Build the application
make build

# For macOS users, create an application bundle
make macapp
```

## Usage

Run the application with:

```bash
make run
```

Or directly:

```bash
./lazyreview
```

## Configuration

LazyReview requires configuration for your GitHub credentials and repositories to monitor. On first run, you'll be prompted to provide these details.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- Built with [Fyne](https://fyne.io/) - Cross platform GUI framework
- Powered by Go
