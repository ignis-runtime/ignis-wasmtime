// Simple TCP client to test the TCP logger server
const net = require('net');

const client = new net.Socket();

client.connect(8080, 'localhost', () => {
    console.log('Connected to TCP server');
    client.write('Hello from TCP client!');
});

client.on('data', (data) => {
    console.log('Received from server: ' + data);
    
    // Send another message after a delay
    setTimeout(() => {
        client.write('Another message from client');
    }, 1000);
});

client.on('close', () => {
    console.log('Connection closed');
});

client.on('error', (err) => {
    console.error('Connection error: ' + err);
});