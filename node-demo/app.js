const express = require('express');
const app = express();
const bodyParser = require('body-parser');
const port = 3000;
app.use(bodyParser.json());
const request = require("request");

// These ports are injected automatically into the container.
const daprPort = process.env.DAPR_HTTP_PORT;
const daprGRPCPort = process.env.DAPR_GRPC_PORT;

// mysql-binding
const bindingsName = `rds-mysql`;
const bindingsUrl = `http://localhost:${daprPort}/v1.0/bindings/${bindingsName}`;


app.get('/', (_req, res) => {
    res.send('Hello Dapr on K8s!');
});

app.get('/order/:id', (req, res) => {
    id = req.params.id;
    console.log(id);
    const query = {
        operation: "query",
        metadata: {
            sql: "SELECT * FROM dapr_bind WHERE id == " + id
        }
    };
    console.log(query.metadata.sql);
    fetch(bindingsUrl, {
        method: "POST",
        body: JSON.stringify(query),
        headers: {
            "Content-Type": "application/json"
        }
    }).then((response) => {
        if (!response.ok) {
            throw "Failed to query data." + response.json();
        }

        console.log("Successfully queried data.");
        res.status(200).send(response.data);
    }).catch((error) => {
        console.log(error);
        res.status(500).send({ message: error });
    });
});




app.post('/neworder', (req, res) => {
    body = JSON.stringify(req.body);
    console.log(body);
    const exec = {
        operation: "exec",
        metadata: {
            sql: "INSERT INTO dapr_bind (customerId,value) VALUES (" + "10020, " + body + ")"
        }
    }
    const options = {
        'method': 'POST',
        'url': bindingsUrl,
        'headers': {
            'Content-Type': 'application/json'
        },
        body: JSON.stringify(exec)
    };

    request(options, function (error, response) {
        if (error) throw new Error(error);
        console.log(response.body);
        res.json(response.body);
    });
});

app.get('/ports', (_req, res) => {
    console.log("DAPR_HTTP_PORT: " + daprPort);
    console.log("DAPR_GRPC_PORT: " + daprGRPCPort);
    res.status(200).send({ DAPR_HTTP_PORT: daprPort, DAPR_GRPC_PORT: daprGRPCPort })
});

app.post('/forward', (req, res) => {
    const options = {
        'method': 'POST',
        'url': req.body.url,
        'headers': {
            'Content-Type': 'application/json'
        },
        body: JSON.stringify(req.body.body)
    };

    request(options, function (error, response) {
        if (error) throw new Error(error);
        console.log(response.body);
        res.json(response.body);
    });
});

app.listen(port, () => {
    console.log(`Example app listening at http://localhost:${port}`)
});