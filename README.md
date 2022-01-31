# ACK service controller for Amazon Relational Database Service (RDS)

This repository contains source code for the AWS Controllers for Kubernetes (ACK) service controller for Amazon RDS.

Please [log issues](https://github.com/aws-controllers-k8s/community/issues) and feedback on the main AWS Controllers for Kubernetes Github project.

## Overview

The ACK service controller for Amazon Relational Database Service (RDS) provides a way to manage RDS database instances directly from Kubernetes. This includes the following database engines:

* [Amazon Aurora](https://aws.amazon.com/rds/aurora/) (MySQL & PostgreSQL)
* [Amazon RDS for PostgreSQL](https://aws.amazon.com/rds/postgresql/)
* [Amazon RDS for MySQL](https://aws.amazon.com/rds/mysql/)
* [Amazon RDS for MariaDB](https://aws.amazon.com/rds/mariadb/)
* [Amazon RDS for Oracle](https://aws.amazon.com/rds/oracle/)
* [Amazon RDS for SQL Server](https://aws.amazon.com/rds/sqlserver/)

The ACK service controller for Amazon RDS provides a set of Kubernetes [custom resource definitions](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/) for interfacing with the [Amazon RDS API](https://docs.aws.amazon.com/AmazonRDS/latest/APIReference/) through a declarative Kubernetes workflow. This lets you to run your applications in Kubernetes with a fully-managed relational database in RDS.

## Getting Started

To learn how to [get started with the ACK service controller for Amazon RDS](https://aws-controllers-k8s.github.io/community/docs/tutorials/rds-example/), please see the [tutorial](https://aws-controllers-k8s.github.io/community/docs/tutorials/rds-example/).

## Help & Feedback

The ACK service controller for Amazon RDS is based on the [Amazon RDS API](https://docs.aws.amazon.com/AmazonRDS/latest/APIReference/). To get a full understanding of how all of the APIs work, please review the [Amazon RDS API documentation](https://docs.aws.amazon.com/AmazonRDS/latest/APIReference/).

You can learn more about [how to use Amazon RDS](https://docs.aws.amazon.com/rds/index.html) through the [documentation](https://docs.aws.amazon.com/rds/index.html).

For [general help with ACK](https://github.com/aws-controllers-k8s/community#help--feedback), please see the [ACK community README](https://github.com/aws-controllers-k8s/community#help--feedback).


## Contributing

We welcome community contributions and pull requests.

See our [contribution guide](https://github.com/aws-controllers-k8s/rds-controller/blob/main/CONTRIBUTING.md) for more information on how to report issues, set up a development environment, and submit code.

We adhere to the [Amazon Open Source Code of Conduct](https://aws.github.io/code-of-conduct).

You can also learn more about our [Governance](https://github.com/aws-controllers-k8s/rds-controller/blob/main/GOVERNANCE.md) structure.

## License

This project is [licensed](https://github.com/aws-controllers-k8s/rds-controller/blob/main/LICENSE) under the Apache-2.0 License.
