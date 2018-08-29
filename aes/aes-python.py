# pip install cryptography
import os
import sys
import base64
from cryptography.hazmat.primitives import hashes
from cryptography.hazmat.primitives.ciphers import Cipher, algorithms, modes
from cryptography.hazmat.backends import default_backend

def encrypt(data, password):
    m = hashes.Hash(hashes.SHA256(), backend=default_backend())
    m.update(password)
    key = m.finalize()

    iv = os.urandom(16)

    backend = default_backend()
    cipher = Cipher(algorithms.AES(key), modes.CFB(iv), backend=backend)
    encryptor = cipher.encryptor()
    encrypted = encryptor.update(data) + encryptor.finalize()
    return base64.b64encode(iv+encrypted)

def decrypt(edata, password):
    edata = base64.b64decode(edata)
    iv = edata[:16]

    m = hashes.Hash(hashes.SHA256(), backend=default_backend())
    m.update(password)
    key = m.finalize()

    backend = default_backend()
    cipher = Cipher(algorithms.AES(key), modes.CFB(iv), backend=backend)
    decryptor = cipher.decryptor()
    dec = decryptor.update(edata[16:]) + decryptor.finalize()
    return  dec

def main():
  password = 'SecretKey'
  plain_text = 'It works!'

  output = encrypt(plain_text, password)
  print(output)
  plaintext = decrypt(output, password)
  print("'" + plaintext + "'")

  plaintext = decrypt("dqlwpamXtpvPnfBXpurYmvbapbs0XSvDAg==", password)
  print("'" + plaintext + "'")

if __name__ == '__main__':
  sys.exit(main())
