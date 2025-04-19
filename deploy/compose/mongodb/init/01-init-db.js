// deploy/compose/mongodb/init/01-init-db.js
db = db.getSiblingDB("qubool_kallyanam");

// Create collections for the chat service
db.createCollection("conversations");
db.createCollection("messages");

// Create indexes
db.conversations.createIndex({ "participants": 1 });
db.messages.createIndex({ "conversation_id": 1, "timestamp": 1 });

// Create an admin user
db.createUser({
  user: "qubool_admin",
  pwd: "qubool123",
  roles: [
    { role: "readWrite", db: "qubool_kallyanam" }
  ]
});