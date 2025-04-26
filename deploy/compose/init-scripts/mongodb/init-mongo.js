// deploy/compose/init-scripts/mongodb/init-mongo.js
db = db.getSiblingDB('admin');

// Create users for each service
db.createUser({
  user: 'chatuser',
  pwd: 'password',
  roles: [
    { role: 'readWrite', db: 'chat' }
  ]
});

db.createUser({
  user: 'userservice',
  pwd: 'password',
  roles: [
    { role: 'readWrite', db: 'user' }
  ]
});

// Create the databases
db = db.getSiblingDB('chat');
db.createCollection('messages');

db = db.getSiblingDB('user');
db.createCollection('profiles');