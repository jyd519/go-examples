'use strict';

const crypto = require('crypto');

const algorithm = 'aes-256-cfb';

function encryptText(text, password) {
  const hash = crypto.createHash('sha256');
  hash.update(password);
  const key = hash.digest();
  const iv = crypto.randomBytes(16);
  const cipher = crypto.createCipheriv(algorithm, key, iv);
  let enc = [iv, cipher.update(text, 'utf8')];
  enc.push(cipher.final());
  return Buffer.concat(enc).toString('base64');
}

function decryptText(text, password) {
  const hash = crypto.createHash('sha256');
  hash.update(password);
  const key = hash.digest();

  const contents = Buffer.from(text, 'base64');
  const iv = contents.slice(0, 16);
  const edata = contents.slice(16);

  const decipher = crypto.createDecipheriv(algorithm, key, iv);
  let res = decipher.update(edata, '', 'utf8');
  res += decipher.final('utf8');

  return res;
}

const encrypted = encryptText('It works!', 'SecretKey');
console.log('Encrypted: ', encrypted);

const decrypted = decryptText(encrypted, 'SecretKey');
console.log('Decrypted: ', decrypted);

const decrypted2 = decryptText('8t1pHqgi/56rM3zg9orwC03d2/wSulMKgg==', 'SecretKey');
console.log('Decrypted: ', decrypted2);


const kTokenKey = 'abcde';

if (process.argv.length > 1) {
  if (process.argv[2] === 'd') {
    const decrypted2 = decryptText(process.argv[3], kTokenKey);
    console.log('Decrypted: ', decrypted2);
  } else {
    const encrypted = encryptText(process.argv[3], kTokenKey);
    console.log('Encrypted: ', encrypted);
  }
}
