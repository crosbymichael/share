var tty = require('tty.js');

var app = tty.createServer({
  shell: 'zsh',
  users: {
    foo: 'bar'
  },
  port: 8000,
  term: {
      "termName": "xterm-256color"
  }
});

app.get('/foo', function(req, res, next) {
  res.send('bar');
});

app.listen();
