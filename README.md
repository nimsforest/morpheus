# Morpheus

Morpheus is a tool designed to help users and organizations plant and grow their Nims Forest. The scaling of the forest happens autonomously through the Nims Forest itself.

## Features

- **Cloud Provider Integration**: Access cloud provider CLIs to launch machines.
- **Docker Integration**: Build and launch the Nims Forest image using Docker CLI.
- **Kubernetes Integration**: Create and manage clusters using Kubernetes CLI.
- **NATS Integration**: Validate if a forest is up and running using NATS CLI.

## Installation

To install Morpheus, clone the repository and install the dependencies:

```bash
git clone https://github.com/yourusername/morpheus.git
cd morpheus
# Install dependencies, e.g., pip install -r requirements.txt if using Python
```

## Usage

Morpheus has a single function: `plant`. This function allows you to plant a Nims Forest in different environments with varying sizes.

### Syntax

```bash
morpheus plant <location> <size> [cloud_provider]
```

### Arguments

- `<location>`: Specifies where the forest needs to be planted. Options:
  - `local`: Runs on the same machine as Morpheus.
  - `on-prem`: Requires the address inside the local network where the forest should be planted.
  - `cloud`: Requires the cloud provider as the second argument.

- `<size>`: Specifies the size of the forest. Options:
  - `wood`: Launches one NATS server.
  - `forest`: Launches a NATS cluster on one machine.
  - `jungle`: Launches a Kubernetes cluster that has a NATS server within.

- `[cloud_provider]`: Specifies the cloud provider when planting in the cloud (required if `location` is `cloud`). Examples: `aws`, `gcp`, `azure`.

### Examples

#### Planting Locally

```bash
morpheus plant local wood
```

This will plant a Nims Forest with a single NATS server on the local machine.

#### Planting On-Premises

```bash
morpheus plant on-prem forest 192.168.1.100
```

This will plant a Nims Forest with a NATS cluster on the specified on-prem address.

#### Planting in the Cloud

```bash
morpheus plant cloud jungle aws
```

This will plant a Nims Forest with a Kubernetes cluster and NATS server on AWS.

## Validation

To validate if the forest is up and running, Morpheus uses the NATS CLI:

```bash
nats server check
```

This command will verify the status of your Nims Forest.

## Contributing

We welcome contributions to Morpheus! Please fork the repository and submit pull requests.

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

---

**Note:** Replace placeholder URLs and paths with actual values. Ensure all dependencies and installation instructions are accurate based on your project's specifics.
