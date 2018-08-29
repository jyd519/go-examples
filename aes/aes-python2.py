from Crypto.Cipher import AES
from Crypto import Random
from Crypto.Hash import SHA256
import M2Crypto.EVP
import base64

password = 'SecretKey'
input = '80aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaIt works!'

BLOCK_SIZE = 16

def pad (data):
    pad = BLOCK_SIZE - len(data) % BLOCK_SIZE
    return data + pad * chr(pad)

def unpad (padded):
    pad = ord(padded[-1])
    return padded[:-pad]

def hex_str(b):
  res = ''
  for d in b:
    res += "%02x" % ord(d)
  return res

def _encrypt(data, nonce, password):
    m = SHA256.new()
    m.update(password)
    key = m.digest()

    # m = md5()
    # m.update(password + key)
    # iv = m.hexdigest()

    rndfile = Random.new()
    iv = rndfile.read(16)

    # data = pad(data)

    aes = AES.new(key, AES.MODE_CFB, iv[:16])
    encrypted = aes.encrypt(data)
    return base64.b64encode(iv+encrypted)

def _decrypt(edata, nonce, password):
    edata = base64.b64decode(edata)

    print(hex_str(edata))

    m = SHA256.new()
    m.update(password)
    key = m.digest()
    # m = md5()
    # m.update(password + key)
    # iv = m.hexdigest()
    iv = edata[:16]

    print("key:", hex_str(key), len(key))
    print("iv:", hex_str(iv))
    print("text:", hex_str(edata[16:]))

    aes = M2Crypto.EVP.Cipher('aes_256_cfb', key, iv, 0)
    dec = aes.update(edata[16:]) + aes.final()

    # aes = AES.new(key, mode=AES.MODE_CFB, IV=iv)
    # return unpad(aes.decrypt(edata))
    # dec = aes.decrypt(edata[16:])

    print("dec:", hex_str(dec))
    return  dec

output = _encrypt(input, "", password)
print(output)
plaintext = _decrypt(output, "", password)
print("'" + plaintext + "'")

plaintext = _decrypt("dqlwpamXtpvPnfBXpurYmvbapbs0XSvDAg==", "", password)
print("'" + plaintext + "'")

