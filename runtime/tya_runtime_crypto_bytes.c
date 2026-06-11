typedef struct {
  uint32_t state[4];
  uint64_t count;
  uint8_t buffer[64];
} tya_md5_ctx;

static void tya_md5_init(tya_md5_ctx *c) {
  c->state[0] = 0x67452301; c->state[1] = 0xEFCDAB89;
  c->state[2] = 0x98BADCFE; c->state[3] = 0x10325476;
  c->count = 0;
}

#define TYA_MD5_F(x, y, z) (((x) & (y)) | (~(x) & (z)))
#define TYA_MD5_G(x, y, z) (((x) & (z)) | ((y) & ~(z)))
#define TYA_MD5_H(x, y, z) ((x) ^ (y) ^ (z))
#define TYA_MD5_I(x, y, z) ((y) ^ ((x) | ~(z)))
#define TYA_MD5_ROL(x, n) (((x) << (n)) | ((x) >> (32 - (n))))
#define TYA_MD5_STEP(f, a, b, c, d, x, t, s) \
  (a) += f((b), (c), (d)) + (x) + (t); \
  (a) = TYA_MD5_ROL((a), (s)); \
  (a) += (b);

static void tya_md5_transform(tya_md5_ctx *ctx, const uint8_t block[64]) {
  uint32_t a = ctx->state[0], b = ctx->state[1], c = ctx->state[2], d = ctx->state[3];
  uint32_t x[16];
  for (int i = 0; i < 16; i++) {
    x[i] = (uint32_t)block[i * 4] | ((uint32_t)block[i * 4 + 1] << 8) |
           ((uint32_t)block[i * 4 + 2] << 16) | ((uint32_t)block[i * 4 + 3] << 24);
  }
  TYA_MD5_STEP(TYA_MD5_F, a, b, c, d, x[ 0], 0xD76AA478,  7)
  TYA_MD5_STEP(TYA_MD5_F, d, a, b, c, x[ 1], 0xE8C7B756, 12)
  TYA_MD5_STEP(TYA_MD5_F, c, d, a, b, x[ 2], 0x242070DB, 17)
  TYA_MD5_STEP(TYA_MD5_F, b, c, d, a, x[ 3], 0xC1BDCEEE, 22)
  TYA_MD5_STEP(TYA_MD5_F, a, b, c, d, x[ 4], 0xF57C0FAF,  7)
  TYA_MD5_STEP(TYA_MD5_F, d, a, b, c, x[ 5], 0x4787C62A, 12)
  TYA_MD5_STEP(TYA_MD5_F, c, d, a, b, x[ 6], 0xA8304613, 17)
  TYA_MD5_STEP(TYA_MD5_F, b, c, d, a, x[ 7], 0xFD469501, 22)
  TYA_MD5_STEP(TYA_MD5_F, a, b, c, d, x[ 8], 0x698098D8,  7)
  TYA_MD5_STEP(TYA_MD5_F, d, a, b, c, x[ 9], 0x8B44F7AF, 12)
  TYA_MD5_STEP(TYA_MD5_F, c, d, a, b, x[10], 0xFFFF5BB1, 17)
  TYA_MD5_STEP(TYA_MD5_F, b, c, d, a, x[11], 0x895CD7BE, 22)
  TYA_MD5_STEP(TYA_MD5_F, a, b, c, d, x[12], 0x6B901122,  7)
  TYA_MD5_STEP(TYA_MD5_F, d, a, b, c, x[13], 0xFD987193, 12)
  TYA_MD5_STEP(TYA_MD5_F, c, d, a, b, x[14], 0xA679438E, 17)
  TYA_MD5_STEP(TYA_MD5_F, b, c, d, a, x[15], 0x49B40821, 22)
  TYA_MD5_STEP(TYA_MD5_G, a, b, c, d, x[ 1], 0xF61E2562,  5)
  TYA_MD5_STEP(TYA_MD5_G, d, a, b, c, x[ 6], 0xC040B340,  9)
  TYA_MD5_STEP(TYA_MD5_G, c, d, a, b, x[11], 0x265E5A51, 14)
  TYA_MD5_STEP(TYA_MD5_G, b, c, d, a, x[ 0], 0xE9B6C7AA, 20)
  TYA_MD5_STEP(TYA_MD5_G, a, b, c, d, x[ 5], 0xD62F105D,  5)
  TYA_MD5_STEP(TYA_MD5_G, d, a, b, c, x[10], 0x02441453,  9)
  TYA_MD5_STEP(TYA_MD5_G, c, d, a, b, x[15], 0xD8A1E681, 14)
  TYA_MD5_STEP(TYA_MD5_G, b, c, d, a, x[ 4], 0xE7D3FBC8, 20)
  TYA_MD5_STEP(TYA_MD5_G, a, b, c, d, x[ 9], 0x21E1CDE6,  5)
  TYA_MD5_STEP(TYA_MD5_G, d, a, b, c, x[14], 0xC33707D6,  9)
  TYA_MD5_STEP(TYA_MD5_G, c, d, a, b, x[ 3], 0xF4D50D87, 14)
  TYA_MD5_STEP(TYA_MD5_G, b, c, d, a, x[ 8], 0x455A14ED, 20)
  TYA_MD5_STEP(TYA_MD5_G, a, b, c, d, x[13], 0xA9E3E905,  5)
  TYA_MD5_STEP(TYA_MD5_G, d, a, b, c, x[ 2], 0xFCEFA3F8,  9)
  TYA_MD5_STEP(TYA_MD5_G, c, d, a, b, x[ 7], 0x676F02D9, 14)
  TYA_MD5_STEP(TYA_MD5_G, b, c, d, a, x[12], 0x8D2A4C8A, 20)
  TYA_MD5_STEP(TYA_MD5_H, a, b, c, d, x[ 5], 0xFFFA3942,  4)
  TYA_MD5_STEP(TYA_MD5_H, d, a, b, c, x[ 8], 0x8771F681, 11)
  TYA_MD5_STEP(TYA_MD5_H, c, d, a, b, x[11], 0x6D9D6122, 16)
  TYA_MD5_STEP(TYA_MD5_H, b, c, d, a, x[14], 0xFDE5380C, 23)
  TYA_MD5_STEP(TYA_MD5_H, a, b, c, d, x[ 1], 0xA4BEEA44,  4)
  TYA_MD5_STEP(TYA_MD5_H, d, a, b, c, x[ 4], 0x4BDECFA9, 11)
  TYA_MD5_STEP(TYA_MD5_H, c, d, a, b, x[ 7], 0xF6BB4B60, 16)
  TYA_MD5_STEP(TYA_MD5_H, b, c, d, a, x[10], 0xBEBFBC70, 23)
  TYA_MD5_STEP(TYA_MD5_H, a, b, c, d, x[13], 0x289B7EC6,  4)
  TYA_MD5_STEP(TYA_MD5_H, d, a, b, c, x[ 0], 0xEAA127FA, 11)
  TYA_MD5_STEP(TYA_MD5_H, c, d, a, b, x[ 3], 0xD4EF3085, 16)
  TYA_MD5_STEP(TYA_MD5_H, b, c, d, a, x[ 6], 0x04881D05, 23)
  TYA_MD5_STEP(TYA_MD5_H, a, b, c, d, x[ 9], 0xD9D4D039,  4)
  TYA_MD5_STEP(TYA_MD5_H, d, a, b, c, x[12], 0xE6DB99E5, 11)
  TYA_MD5_STEP(TYA_MD5_H, c, d, a, b, x[15], 0x1FA27CF8, 16)
  TYA_MD5_STEP(TYA_MD5_H, b, c, d, a, x[ 2], 0xC4AC5665, 23)
  TYA_MD5_STEP(TYA_MD5_I, a, b, c, d, x[ 0], 0xF4292244,  6)
  TYA_MD5_STEP(TYA_MD5_I, d, a, b, c, x[ 7], 0x432AFF97, 10)
  TYA_MD5_STEP(TYA_MD5_I, c, d, a, b, x[14], 0xAB9423A7, 15)
  TYA_MD5_STEP(TYA_MD5_I, b, c, d, a, x[ 5], 0xFC93A039, 21)
  TYA_MD5_STEP(TYA_MD5_I, a, b, c, d, x[12], 0x655B59C3,  6)
  TYA_MD5_STEP(TYA_MD5_I, d, a, b, c, x[ 3], 0x8F0CCC92, 10)
  TYA_MD5_STEP(TYA_MD5_I, c, d, a, b, x[10], 0xFFEFF47D, 15)
  TYA_MD5_STEP(TYA_MD5_I, b, c, d, a, x[ 1], 0x85845DD1, 21)
  TYA_MD5_STEP(TYA_MD5_I, a, b, c, d, x[ 8], 0x6FA87E4F,  6)
  TYA_MD5_STEP(TYA_MD5_I, d, a, b, c, x[15], 0xFE2CE6E0, 10)
  TYA_MD5_STEP(TYA_MD5_I, c, d, a, b, x[ 6], 0xA3014314, 15)
  TYA_MD5_STEP(TYA_MD5_I, b, c, d, a, x[13], 0x4E0811A1, 21)
  TYA_MD5_STEP(TYA_MD5_I, a, b, c, d, x[ 4], 0xF7537E82,  6)
  TYA_MD5_STEP(TYA_MD5_I, d, a, b, c, x[11], 0xBD3AF235, 10)
  TYA_MD5_STEP(TYA_MD5_I, c, d, a, b, x[ 2], 0x2AD7D2BB, 15)
  TYA_MD5_STEP(TYA_MD5_I, b, c, d, a, x[ 9], 0xEB86D391, 21)
  ctx->state[0] += a; ctx->state[1] += b;
  ctx->state[2] += c; ctx->state[3] += d;
}

static void tya_md5_update(tya_md5_ctx *c, const uint8_t *data, size_t len) {
  size_t buf_used = (size_t)((c->count >> 3) & 0x3F);
  c->count += (uint64_t)len << 3;
  size_t need = 64 - buf_used;
  if (len >= need) {
    memcpy(c->buffer + buf_used, data, need);
    tya_md5_transform(c, c->buffer);
    data += need; len -= need;
    while (len >= 64) {
      tya_md5_transform(c, data);
      data += 64; len -= 64;
    }
    buf_used = 0;
  }
  memcpy(c->buffer + buf_used, data, len);
}

static void tya_md5_final(tya_md5_ctx *c, uint8_t out[16]) {
  size_t buf_used = (size_t)((c->count >> 3) & 0x3F);
  c->buffer[buf_used++] = 0x80;
  if (buf_used > 56) {
    memset(c->buffer + buf_used, 0, 64 - buf_used);
    tya_md5_transform(c, c->buffer);
    buf_used = 0;
  }
  memset(c->buffer + buf_used, 0, 56 - buf_used);
  for (int i = 0; i < 8; i++) {
    c->buffer[56 + i] = (uint8_t)((c->count >> (i * 8)) & 0xFF);
  }
  tya_md5_transform(c, c->buffer);
  for (int i = 0; i < 4; i++) {
    out[i * 4] = (uint8_t)(c->state[i] & 0xFF);
    out[i * 4 + 1] = (uint8_t)((c->state[i] >> 8) & 0xFF);
    out[i * 4 + 2] = (uint8_t)((c->state[i] >> 16) & 0xFF);
    out[i * 4 + 3] = (uint8_t)((c->state[i] >> 24) & 0xFF);
  }
}

/* ---- SHA1 ---- */
typedef struct {
  uint32_t state[5];
  uint64_t count;
  uint8_t buffer[64];
} tya_sha1_ctx;

static void tya_sha1_init(tya_sha1_ctx *c) {
  c->state[0] = 0x67452301; c->state[1] = 0xEFCDAB89;
  c->state[2] = 0x98BADCFE; c->state[3] = 0x10325476;
  c->state[4] = 0xC3D2E1F0;
  c->count = 0;
}

#define TYA_SHA1_ROL(x, n) (((x) << (n)) | ((x) >> (32 - (n))))

static void tya_sha1_transform(tya_sha1_ctx *ctx, const uint8_t block[64]) {
  uint32_t w[80];
  for (int i = 0; i < 16; i++) {
    w[i] = ((uint32_t)block[i * 4] << 24) | ((uint32_t)block[i * 4 + 1] << 16) |
           ((uint32_t)block[i * 4 + 2] << 8) | (uint32_t)block[i * 4 + 3];
  }
  for (int i = 16; i < 80; i++) {
    w[i] = TYA_SHA1_ROL(w[i - 3] ^ w[i - 8] ^ w[i - 14] ^ w[i - 16], 1);
  }
  uint32_t a = ctx->state[0], b = ctx->state[1], c = ctx->state[2], d = ctx->state[3], e = ctx->state[4];
  for (int i = 0; i < 80; i++) {
    uint32_t f, k;
    if (i < 20) { f = (b & c) | (~b & d); k = 0x5A827999; }
    else if (i < 40) { f = b ^ c ^ d; k = 0x6ED9EBA1; }
    else if (i < 60) { f = (b & c) | (b & d) | (c & d); k = 0x8F1BBCDC; }
    else { f = b ^ c ^ d; k = 0xCA62C1D6; }
    uint32_t t = TYA_SHA1_ROL(a, 5) + f + e + k + w[i];
    e = d; d = c; c = TYA_SHA1_ROL(b, 30); b = a; a = t;
  }
  ctx->state[0] += a; ctx->state[1] += b;
  ctx->state[2] += c; ctx->state[3] += d;
  ctx->state[4] += e;
}

static void tya_sha1_update(tya_sha1_ctx *c, const uint8_t *data, size_t len) {
  size_t buf_used = (size_t)((c->count >> 3) & 0x3F);
  c->count += (uint64_t)len << 3;
  size_t need = 64 - buf_used;
  if (len >= need) {
    memcpy(c->buffer + buf_used, data, need);
    tya_sha1_transform(c, c->buffer);
    data += need; len -= need;
    while (len >= 64) {
      tya_sha1_transform(c, data);
      data += 64; len -= 64;
    }
    buf_used = 0;
  }
  memcpy(c->buffer + buf_used, data, len);
}

static void tya_sha1_final(tya_sha1_ctx *c, uint8_t out[20]) {
  size_t buf_used = (size_t)((c->count >> 3) & 0x3F);
  c->buffer[buf_used++] = 0x80;
  if (buf_used > 56) {
    memset(c->buffer + buf_used, 0, 64 - buf_used);
    tya_sha1_transform(c, c->buffer);
    buf_used = 0;
  }
  memset(c->buffer + buf_used, 0, 56 - buf_used);
  for (int i = 0; i < 8; i++) {
    c->buffer[56 + i] = (uint8_t)((c->count >> (56 - i * 8)) & 0xFF);
  }
  tya_sha1_transform(c, c->buffer);
  for (int i = 0; i < 5; i++) {
    out[i * 4] = (uint8_t)((c->state[i] >> 24) & 0xFF);
    out[i * 4 + 1] = (uint8_t)((c->state[i] >> 16) & 0xFF);
    out[i * 4 + 2] = (uint8_t)((c->state[i] >> 8) & 0xFF);
    out[i * 4 + 3] = (uint8_t)(c->state[i] & 0xFF);
  }
}

/* ---- SHA-256 ---- */
typedef struct {
  uint32_t state[8];
  uint64_t count;
  uint8_t buffer[64];
} tya_sha256_ctx;

static const uint32_t tya_sha256_k[64] = {
  0x428a2f98, 0x71374491, 0xb5c0fbcf, 0xe9b5dba5, 0x3956c25b, 0x59f111f1, 0x923f82a4, 0xab1c5ed5,
  0xd807aa98, 0x12835b01, 0x243185be, 0x550c7dc3, 0x72be5d74, 0x80deb1fe, 0x9bdc06a7, 0xc19bf174,
  0xe49b69c1, 0xefbe4786, 0x0fc19dc6, 0x240ca1cc, 0x2de92c6f, 0x4a7484aa, 0x5cb0a9dc, 0x76f988da,
  0x983e5152, 0xa831c66d, 0xb00327c8, 0xbf597fc7, 0xc6e00bf3, 0xd5a79147, 0x06ca6351, 0x14292967,
  0x27b70a85, 0x2e1b2138, 0x4d2c6dfc, 0x53380d13, 0x650a7354, 0x766a0abb, 0x81c2c92e, 0x92722c85,
  0xa2bfe8a1, 0xa81a664b, 0xc24b8b70, 0xc76c51a3, 0xd192e819, 0xd6990624, 0xf40e3585, 0x106aa070,
  0x19a4c116, 0x1e376c08, 0x2748774c, 0x34b0bcb5, 0x391c0cb3, 0x4ed8aa4a, 0x5b9cca4f, 0x682e6ff3,
  0x748f82ee, 0x78a5636f, 0x84c87814, 0x8cc70208, 0x90befffa, 0xa4506ceb, 0xbef9a3f7, 0xc67178f2,
};

static void tya_sha256_init(tya_sha256_ctx *c) {
  c->state[0] = 0x6a09e667; c->state[1] = 0xbb67ae85;
  c->state[2] = 0x3c6ef372; c->state[3] = 0xa54ff53a;
  c->state[4] = 0x510e527f; c->state[5] = 0x9b05688c;
  c->state[6] = 0x1f83d9ab; c->state[7] = 0x5be0cd19;
  c->count = 0;
}

#define TYA_SHA256_ROR(x, n) (((x) >> (n)) | ((x) << (32 - (n))))

static void tya_sha256_transform(tya_sha256_ctx *ctx, const uint8_t block[64]) {
  uint32_t w[64];
  for (int i = 0; i < 16; i++) {
    w[i] = ((uint32_t)block[i * 4] << 24) | ((uint32_t)block[i * 4 + 1] << 16) |
           ((uint32_t)block[i * 4 + 2] << 8) | (uint32_t)block[i * 4 + 3];
  }
  for (int i = 16; i < 64; i++) {
    uint32_t s0 = TYA_SHA256_ROR(w[i - 15], 7) ^ TYA_SHA256_ROR(w[i - 15], 18) ^ (w[i - 15] >> 3);
    uint32_t s1 = TYA_SHA256_ROR(w[i - 2], 17) ^ TYA_SHA256_ROR(w[i - 2], 19) ^ (w[i - 2] >> 10);
    w[i] = w[i - 16] + s0 + w[i - 7] + s1;
  }
  uint32_t a = ctx->state[0], b = ctx->state[1], c = ctx->state[2], d = ctx->state[3];
  uint32_t e = ctx->state[4], f = ctx->state[5], g = ctx->state[6], h = ctx->state[7];
  for (int i = 0; i < 64; i++) {
    uint32_t S1 = TYA_SHA256_ROR(e, 6) ^ TYA_SHA256_ROR(e, 11) ^ TYA_SHA256_ROR(e, 25);
    uint32_t ch = (e & f) ^ (~e & g);
    uint32_t t1 = h + S1 + ch + tya_sha256_k[i] + w[i];
    uint32_t S0 = TYA_SHA256_ROR(a, 2) ^ TYA_SHA256_ROR(a, 13) ^ TYA_SHA256_ROR(a, 22);
    uint32_t mj = (a & b) ^ (a & c) ^ (b & c);
    uint32_t t2 = S0 + mj;
    h = g; g = f; f = e; e = d + t1;
    d = c; c = b; b = a; a = t1 + t2;
  }
  ctx->state[0] += a; ctx->state[1] += b;
  ctx->state[2] += c; ctx->state[3] += d;
  ctx->state[4] += e; ctx->state[5] += f;
  ctx->state[6] += g; ctx->state[7] += h;
}

static void tya_sha256_update(tya_sha256_ctx *c, const uint8_t *data, size_t len) {
  size_t buf_used = (size_t)((c->count >> 3) & 0x3F);
  c->count += (uint64_t)len << 3;
  size_t need = 64 - buf_used;
  if (len >= need) {
    memcpy(c->buffer + buf_used, data, need);
    tya_sha256_transform(c, c->buffer);
    data += need; len -= need;
    while (len >= 64) {
      tya_sha256_transform(c, data);
      data += 64; len -= 64;
    }
    buf_used = 0;
  }
  memcpy(c->buffer + buf_used, data, len);
}

static void tya_sha256_final(tya_sha256_ctx *c, uint8_t out[32]) {
  size_t buf_used = (size_t)((c->count >> 3) & 0x3F);
  c->buffer[buf_used++] = 0x80;
  if (buf_used > 56) {
    memset(c->buffer + buf_used, 0, 64 - buf_used);
    tya_sha256_transform(c, c->buffer);
    buf_used = 0;
  }
  memset(c->buffer + buf_used, 0, 56 - buf_used);
  for (int i = 0; i < 8; i++) {
    c->buffer[56 + i] = (uint8_t)((c->count >> (56 - i * 8)) & 0xFF);
  }
  tya_sha256_transform(c, c->buffer);
  for (int i = 0; i < 8; i++) {
    out[i * 4] = (uint8_t)((c->state[i] >> 24) & 0xFF);
    out[i * 4 + 1] = (uint8_t)((c->state[i] >> 16) & 0xFF);
    out[i * 4 + 2] = (uint8_t)((c->state[i] >> 8) & 0xFF);
    out[i * 4 + 3] = (uint8_t)(c->state[i] & 0xFF);
  }
}

/* ---- SHA-512 (and SHA-384) ---- */
typedef struct {
  uint64_t state[8];
  uint64_t count_lo;
  uint64_t count_hi;
  uint8_t buffer[128];
} tya_sha512_ctx;

static const uint64_t tya_sha512_k[80] = {
  0x428a2f98d728ae22ULL, 0x7137449123ef65cdULL, 0xb5c0fbcfec4d3b2fULL, 0xe9b5dba58189dbbcULL,
  0x3956c25bf348b538ULL, 0x59f111f1b605d019ULL, 0x923f82a4af194f9bULL, 0xab1c5ed5da6d8118ULL,
  0xd807aa98a3030242ULL, 0x12835b0145706fbeULL, 0x243185be4ee4b28cULL, 0x550c7dc3d5ffb4e2ULL,
  0x72be5d74f27b896fULL, 0x80deb1fe3b1696b1ULL, 0x9bdc06a725c71235ULL, 0xc19bf174cf692694ULL,
  0xe49b69c19ef14ad2ULL, 0xefbe4786384f25e3ULL, 0x0fc19dc68b8cd5b5ULL, 0x240ca1cc77ac9c65ULL,
  0x2de92c6f592b0275ULL, 0x4a7484aa6ea6e483ULL, 0x5cb0a9dcbd41fbd4ULL, 0x76f988da831153b5ULL,
  0x983e5152ee66dfabULL, 0xa831c66d2db43210ULL, 0xb00327c898fb213fULL, 0xbf597fc7beef0ee4ULL,
  0xc6e00bf33da88fc2ULL, 0xd5a79147930aa725ULL, 0x06ca6351e003826fULL, 0x142929670a0e6e70ULL,
  0x27b70a8546d22ffcULL, 0x2e1b21385c26c926ULL, 0x4d2c6dfc5ac42aedULL, 0x53380d139d95b3dfULL,
  0x650a73548baf63deULL, 0x766a0abb3c77b2a8ULL, 0x81c2c92e47edaee6ULL, 0x92722c851482353bULL,
  0xa2bfe8a14cf10364ULL, 0xa81a664bbc423001ULL, 0xc24b8b70d0f89791ULL, 0xc76c51a30654be30ULL,
  0xd192e819d6ef5218ULL, 0xd69906245565a910ULL, 0xf40e35855771202aULL, 0x106aa07032bbd1b8ULL,
  0x19a4c116b8d2d0c8ULL, 0x1e376c085141ab53ULL, 0x2748774cdf8eeb99ULL, 0x34b0bcb5e19b48a8ULL,
  0x391c0cb3c5c95a63ULL, 0x4ed8aa4ae3418acbULL, 0x5b9cca4f7763e373ULL, 0x682e6ff3d6b2b8a3ULL,
  0x748f82ee5defb2fcULL, 0x78a5636f43172f60ULL, 0x84c87814a1f0ab72ULL, 0x8cc702081a6439ecULL,
  0x90befffa23631e28ULL, 0xa4506cebde82bde9ULL, 0xbef9a3f7b2c67915ULL, 0xc67178f2e372532bULL,
  0xca273eceea26619cULL, 0xd186b8c721c0c207ULL, 0xeada7dd6cde0eb1eULL, 0xf57d4f7fee6ed178ULL,
  0x06f067aa72176fbaULL, 0x0a637dc5a2c898a6ULL, 0x113f9804bef90daeULL, 0x1b710b35131c471bULL,
  0x28db77f523047d84ULL, 0x32caab7b40c72493ULL, 0x3c9ebe0a15c9bebcULL, 0x431d67c49c100d4cULL,
  0x4cc5d4becb3e42b6ULL, 0x597f299cfc657e2aULL, 0x5fcb6fab3ad6faecULL, 0x6c44198c4a475817ULL,
};

#define TYA_SHA512_ROR(x, n) (((x) >> (n)) | ((x) << (64 - (n))))

static void tya_sha512_transform(tya_sha512_ctx *ctx, const uint8_t block[128]) {
  uint64_t w[80];
  for (int i = 0; i < 16; i++) {
    w[i] = 0;
    for (int j = 0; j < 8; j++) {
      w[i] = (w[i] << 8) | block[i * 8 + j];
    }
  }
  for (int i = 16; i < 80; i++) {
    uint64_t s0 = TYA_SHA512_ROR(w[i - 15], 1) ^ TYA_SHA512_ROR(w[i - 15], 8) ^ (w[i - 15] >> 7);
    uint64_t s1 = TYA_SHA512_ROR(w[i - 2], 19) ^ TYA_SHA512_ROR(w[i - 2], 61) ^ (w[i - 2] >> 6);
    w[i] = w[i - 16] + s0 + w[i - 7] + s1;
  }
  uint64_t a = ctx->state[0], b = ctx->state[1], c = ctx->state[2], d = ctx->state[3];
  uint64_t e = ctx->state[4], f = ctx->state[5], g = ctx->state[6], h = ctx->state[7];
  for (int i = 0; i < 80; i++) {
    uint64_t S1 = TYA_SHA512_ROR(e, 14) ^ TYA_SHA512_ROR(e, 18) ^ TYA_SHA512_ROR(e, 41);
    uint64_t ch = (e & f) ^ (~e & g);
    uint64_t t1 = h + S1 + ch + tya_sha512_k[i] + w[i];
    uint64_t S0 = TYA_SHA512_ROR(a, 28) ^ TYA_SHA512_ROR(a, 34) ^ TYA_SHA512_ROR(a, 39);
    uint64_t mj = (a & b) ^ (a & c) ^ (b & c);
    uint64_t t2 = S0 + mj;
    h = g; g = f; f = e; e = d + t1;
    d = c; c = b; b = a; a = t1 + t2;
  }
  ctx->state[0] += a; ctx->state[1] += b;
  ctx->state[2] += c; ctx->state[3] += d;
  ctx->state[4] += e; ctx->state[5] += f;
  ctx->state[6] += g; ctx->state[7] += h;
}

static void tya_sha512_init(tya_sha512_ctx *c) {
  c->state[0] = 0x6a09e667f3bcc908ULL; c->state[1] = 0xbb67ae8584caa73bULL;
  c->state[2] = 0x3c6ef372fe94f82bULL; c->state[3] = 0xa54ff53a5f1d36f1ULL;
  c->state[4] = 0x510e527fade682d1ULL; c->state[5] = 0x9b05688c2b3e6c1fULL;
  c->state[6] = 0x1f83d9abfb41bd6bULL; c->state[7] = 0x5be0cd19137e2179ULL;
  c->count_lo = 0; c->count_hi = 0;
}

static void tya_sha384_init(tya_sha512_ctx *c) {
  c->state[0] = 0xcbbb9d5dc1059ed8ULL; c->state[1] = 0x629a292a367cd507ULL;
  c->state[2] = 0x9159015a3070dd17ULL; c->state[3] = 0x152fecd8f70e5939ULL;
  c->state[4] = 0x67332667ffc00b31ULL; c->state[5] = 0x8eb44a8768581511ULL;
  c->state[6] = 0xdb0c2e0d64f98fa7ULL; c->state[7] = 0x47b5481dbefa4fa4ULL;
  c->count_lo = 0; c->count_hi = 0;
}

static void tya_sha512_update(tya_sha512_ctx *c, const uint8_t *data, size_t len) {
  size_t buf_used = (size_t)((c->count_lo >> 3) & 0x7F);
  uint64_t add = (uint64_t)len << 3;
  uint64_t old_lo = c->count_lo;
  c->count_lo += add;
  if (c->count_lo < old_lo) c->count_hi++;
  c->count_hi += (uint64_t)len >> 61;
  size_t need = 128 - buf_used;
  if (len >= need) {
    memcpy(c->buffer + buf_used, data, need);
    tya_sha512_transform(c, c->buffer);
    data += need; len -= need;
    while (len >= 128) {
      tya_sha512_transform(c, data);
      data += 128; len -= 128;
    }
    buf_used = 0;
  }
  memcpy(c->buffer + buf_used, data, len);
}

static void tya_sha512_final_n(tya_sha512_ctx *c, uint8_t *out, int out_words) {
  size_t buf_used = (size_t)((c->count_lo >> 3) & 0x7F);
  c->buffer[buf_used++] = 0x80;
  if (buf_used > 112) {
    memset(c->buffer + buf_used, 0, 128 - buf_used);
    tya_sha512_transform(c, c->buffer);
    buf_used = 0;
  }
  memset(c->buffer + buf_used, 0, 112 - buf_used);
  for (int i = 0; i < 8; i++) {
    c->buffer[112 + i] = (uint8_t)((c->count_hi >> (56 - i * 8)) & 0xFF);
  }
  for (int i = 0; i < 8; i++) {
    c->buffer[120 + i] = (uint8_t)((c->count_lo >> (56 - i * 8)) & 0xFF);
  }
  tya_sha512_transform(c, c->buffer);
  for (int i = 0; i < out_words; i++) {
    for (int j = 0; j < 8; j++) {
      out[i * 8 + j] = (uint8_t)((c->state[i] >> (56 - j * 8)) & 0xFF);
    }
  }
}

static const char tya_hex_digits[] = "0123456789abcdef";

static TyaValue tya_hex_string(const uint8_t *data, size_t n) {
  char *out = malloc(n * 2 + 1);
  for (size_t i = 0; i < n; i++) {
    out[i * 2] = tya_hex_digits[(data[i] >> 4) & 0xF];
    out[i * 2 + 1] = tya_hex_digits[data[i] & 0xF];
  }
  out[n * 2] = '\0';
  return tya_string(out);
}

TyaValue tya_digest_md5(TyaValue text) {
  const uint8_t *data;
  size_t dlen;
  if (text.kind == TYA_STRING && text.string != NULL) {
    data = (const uint8_t *)text.string;
    dlen = strlen(text.string);
  } else if (text.kind == TYA_BYTES && text.bytes != NULL) {
    data = text.bytes->data;
    dlen = (size_t)text.bytes->len;
  } else {
    tya_raise(tya_string("digest.md5: argument must be a string or bytes"));
    return tya_nil();
  }
  tya_md5_ctx c;
  tya_md5_init(&c);
  tya_md5_update(&c, data, dlen);
  uint8_t digest[16];
  tya_md5_final(&c, digest);
  return tya_hex_string(digest, 16);
}

static int tya_digest_input(TyaValue v, const uint8_t **data, size_t *dlen, const char *err_msg) {
  if (v.kind == TYA_STRING && v.string != NULL) {
    *data = (const uint8_t *)v.string;
    *dlen = strlen(v.string);
    return 0;
  }
  if (v.kind == TYA_BYTES && v.bytes != NULL) {
    *data = v.bytes->data;
    *dlen = (size_t)v.bytes->len;
    return 0;
  }
  tya_raise(tya_string(err_msg));
  return -1;
}

TyaValue tya_digest_sha1(TyaValue text) {
  const uint8_t *data;
  size_t dlen;
  if (tya_digest_input(text, &data, &dlen, "digest.sha1: argument must be a string or bytes") < 0) {
    return tya_nil();
  }
  tya_sha1_ctx c;
  tya_sha1_init(&c);
  tya_sha1_update(&c, data, dlen);
  uint8_t digest[20];
  tya_sha1_final(&c, digest);
  return tya_hex_string(digest, 20);
}

TyaValue tya_digest_sha256(TyaValue text) {
  const uint8_t *data;
  size_t dlen;
  if (tya_digest_input(text, &data, &dlen, "digest.sha256: argument must be a string or bytes") < 0) {
    return tya_nil();
  }
  tya_sha256_ctx c;
  tya_sha256_init(&c);
  tya_sha256_update(&c, data, dlen);
  uint8_t digest[32];
  tya_sha256_final(&c, digest);
  return tya_hex_string(digest, 32);
}

TyaValue tya_digest_sha384(TyaValue text) {
  const uint8_t *data;
  size_t dlen;
  if (tya_digest_input(text, &data, &dlen, "digest.sha384: argument must be a string or bytes") < 0) {
    return tya_nil();
  }
  tya_sha512_ctx c;
  tya_sha384_init(&c);
  tya_sha512_update(&c, data, dlen);
  uint8_t digest[48];
  tya_sha512_final_n(&c, digest, 6);
  return tya_hex_string(digest, 48);
}

TyaValue tya_digest_sha512(TyaValue text) {
  const uint8_t *data;
  size_t dlen;
  if (tya_digest_input(text, &data, &dlen, "digest.sha512: argument must be a string or bytes") < 0) {
    return tya_nil();
  }
  tya_sha512_ctx c;
  tya_sha512_init(&c);
  tya_sha512_update(&c, data, dlen);
  uint8_t digest[64];
  tya_sha512_final_n(&c, digest, 8);
  return tya_hex_string(digest, 64);
}

/* =========================================================================
 * v0.24: secure_random
 * ========================================================================= */

static int tya_secure_random_fill(uint8_t *buf, size_t n) {
#if defined(__APPLE__) || defined(__FreeBSD__) || defined(__OpenBSD__)
  while (n > 0) {
    size_t chunk = n > 256 ? 256 : n;
    if (getentropy(buf, chunk) < 0) return -1;
    buf += chunk; n -= chunk;
  }
  return 0;
#else
  int fd = open("/dev/urandom", O_RDONLY);
  if (fd < 0) return -1;
  while (n > 0) {
    ssize_t r = read(fd, buf, n);
    if (r < 0) {
      if (errno == EINTR) continue;
      close(fd);
      return -1;
    }
    if (r == 0) { close(fd); return -1; }
    buf += r; n -= (size_t)r;
  }
  close(fd);
  return 0;
#endif
}

TyaValue tya_secure_random_bytes(TyaValue n) {
  if (n.kind != TYA_NUMBER) {
    tya_raise(tya_string("secure_random: count must be a number"));
    return tya_nil();
  }
  long count = (long)n.number;
  if (count < 0 || count > 4096) {
    tya_raise(tya_string("secure_random: count out of range"));
    return tya_nil();
  }
  TyaBytes *bb = tya_gc_alloc(sizeof(TyaBytes), TYA_GC_BYTES);
  bb->len = (int)count;
  bb->data = malloc((size_t)(count > 0 ? count : 1));
  if (tya_secure_random_fill(bb->data, (size_t)count) < 0) {
    free(bb->data);
    /* bb is GC-tracked; leak now, the next collection will reclaim it. */
    tya_raise(tya_string("secure_random: entropy source unavailable"));
    return tya_nil();
  }
  return (TyaValue){.kind = TYA_BYTES, .bytes = bb};
}

TyaValue tya_secure_random_int(TyaValue min, TyaValue max) {
  if (min.kind != TYA_NUMBER || max.kind != TYA_NUMBER) {
    tya_raise(tya_string("secure_random.int: arguments must be numbers"));
    return tya_nil();
  }
  long mn = (long)min.number;
  long mx = (long)max.number;
  if (mx < mn) {
    tya_raise(tya_string("secure_random.int: max < min"));
    return tya_nil();
  }
  uint64_t range = (uint64_t)(mx - mn) + 1ULL;
  uint64_t threshold = (uint64_t)(-(int64_t)range) % range;
  for (;;) {
    uint64_t r;
    if (tya_secure_random_fill((uint8_t *)&r, sizeof(r)) < 0) {
      tya_raise(tya_string("secure_random.int: entropy source unavailable"));
      return tya_nil();
    }
    if (r >= threshold) {
      return tya_number((double)(long)((r % range) + (uint64_t)mn));
    }
  }
}

/* =========================================================================
 * v0.25: bytes type and binary I/O
 * ========================================================================= */

TyaValue tya_bytes_lit(const char *data, int len) {
  TyaBytes *b = tya_gc_alloc(sizeof(TyaBytes), TYA_GC_BYTES);
  b->len = len;
  b->data = malloc((size_t)(len > 0 ? len : 1));
  if (len > 0) memcpy(b->data, data, (size_t)len);
  return (TyaValue){.kind = TYA_BYTES, .bytes = b};
}

TyaValue tya_bytes_from_array(TyaValue arr) {
  if (arr.kind != TYA_ARRAY || arr.array == NULL) {
    tya_raise(tya_string("bytes: argument must be an array of ints"));
    return tya_nil();
  }
  int n = arr.array->len;
  TyaBytes *b = tya_gc_alloc(sizeof(TyaBytes), TYA_GC_BYTES);
  b->len = n;
  b->data = malloc((size_t)(n > 0 ? n : 1));
  for (int i = 0; i < n; i++) {
    TyaValue item = arr.array->items[i];
    if (item.kind != TYA_NUMBER) {
      free(b->data);
      /* b is GC-tracked; leak now, the next collection will reclaim it. */
      tya_raise(tya_string("bytes: items must be ints"));
      return tya_nil();
    }
    int v = (int)item.number;
    if (v < 0 || v > 255) {
      free(b->data);
      /* b is GC-tracked; leak now, the next collection will reclaim it. */
      tya_raise(tya_string("bytes: item out of 0..255"));
      return tya_nil();
    }
    b->data[i] = (uint8_t)v;
  }
  return (TyaValue){.kind = TYA_BYTES, .bytes = b};
}

TyaValue tya_bytes_of(TyaValue text) {
  if (text.kind != TYA_STRING || text.string == NULL) {
    tya_raise(tya_string("bytes_of: argument must be a string"));
    return tya_nil();
  }
  int n = (int)strlen(text.string);
  return tya_bytes_lit(text.string, n);
}

TyaValue tya_bytes_text(TyaValue b) {
  if (b.kind != TYA_BYTES || b.bytes == NULL) {
    tya_raise(tya_string("bytes_text: argument must be bytes"));
    return tya_nil();
  }
  for (int i = 0; i < b.bytes->len; i++) {
    if (b.bytes->data[i] == 0) {
      tya_raise(tya_string("bytes_text: NUL byte not allowed in string"));
      return tya_nil();
    }
  }
  if (!tya_utf8_valid_bytes((const unsigned char *)b.bytes->data, b.bytes->len)) {
    tya_raise(tya_string("bytes_text: invalid UTF-8"));
    return tya_nil();
  }
  char *out = malloc((size_t)b.bytes->len + 1);
  memcpy(out, b.bytes->data, (size_t)b.bytes->len);
  out[b.bytes->len] = '\0';
  return tya_string(out);
}

TyaValue tya_bytes_array(TyaValue b) {
  if (b.kind != TYA_BYTES || b.bytes == NULL) {
    tya_raise(tya_string("bytes_array: argument must be bytes"));
    return tya_nil();
  }
  TyaValue out = tya_array(NULL, 0);
  for (int i = 0; i < b.bytes->len; i++) {
    tya_push(out, tya_number((double)b.bytes->data[i]));
  }
  return out;
}

TyaValue tya_bytes_concat(TyaValue a, TyaValue b) {
  if (a.kind != TYA_BYTES || b.kind != TYA_BYTES || a.bytes == NULL || b.bytes == NULL) {
    tya_raise(tya_string("bytes_concat: arguments must be bytes"));
    return tya_nil();
  }
  int total = a.bytes->len + b.bytes->len;
  TyaBytes *out = tya_gc_alloc(sizeof(TyaBytes), TYA_GC_BYTES);
  out->len = total;
  out->data = malloc((size_t)(total > 0 ? total : 1));
  if (a.bytes->len > 0) memcpy(out->data, a.bytes->data, (size_t)a.bytes->len);
  if (b.bytes->len > 0) memcpy(out->data + a.bytes->len, b.bytes->data, (size_t)b.bytes->len);
  return (TyaValue){.kind = TYA_BYTES, .bytes = out};
}

TyaValue tya_bytes_slice(TyaValue b, TyaValue start_v, TyaValue end_v) {
  if (b.kind != TYA_BYTES || b.bytes == NULL) {
    tya_raise(tya_string("bytes_slice: first argument must be bytes"));
    return tya_nil();
  }
  if (start_v.kind != TYA_NUMBER || end_v.kind != TYA_NUMBER) {
    tya_raise(tya_string("bytes_slice: indices must be ints"));
    return tya_nil();
  }
  int s = (int)start_v.number;
  int e = (int)end_v.number;
  if (s < 0 || e < s || e > b.bytes->len) {
    tya_raise(tya_string("bytes_slice: index out of range"));
    return tya_nil();
  }
  return tya_bytes_lit((const char *)(b.bytes->data + s), e - s);
}

TyaValue tya_file_read_bytes(TyaValue path) {
  if (path.kind != TYA_STRING || path.string == NULL) {
    tya_raise(tya_string("file.read_bytes: path must be a string"));
    return tya_nil();
  }
  FILE *fp = fopen(path.string, "rb");
  if (fp == NULL) {
    tya_raise(tya_string(strerror(errno)));
    return tya_nil();
  }
  fseek(fp, 0, SEEK_END);
  long size = ftell(fp);
  fseek(fp, 0, SEEK_SET);
  if (size < 0) size = 0;
  TyaBytes *bb = tya_gc_alloc(sizeof(TyaBytes), TYA_GC_BYTES);
  bb->len = (int)size;
  bb->data = malloc((size_t)(size > 0 ? size : 1));
  size_t got = fread(bb->data, 1, (size_t)size, fp);
  fclose(fp);
  bb->len = (int)got;
  return (TyaValue){.kind = TYA_BYTES, .bytes = bb};
}

TyaValue tya_file_write_bytes(TyaValue path, TyaValue b) {
  if (path.kind != TYA_STRING || path.string == NULL) {
    tya_raise(tya_string("file.write_bytes: path must be a string"));
    return tya_nil();
  }
  if (b.kind != TYA_BYTES || b.bytes == NULL) {
    tya_raise(tya_string("file.write_bytes: data must be bytes"));
    return tya_nil();
  }
  FILE *fp = fopen(path.string, "wb");
  if (fp == NULL) {
    tya_raise(tya_string(strerror(errno)));
    return tya_nil();
  }
  if (b.bytes->len > 0) {
    fwrite(b.bytes->data, 1, (size_t)b.bytes->len, fp);
  }
  fclose(fp);
  return tya_nil();
}

static bool tya_binary_little(TyaValue endian) {
  return endian.kind == TYA_STRING && endian.string != NULL && strcmp(endian.string, "little") == 0;
}

static uint32_t tya_binary_u32(TyaValue b, int offset, bool little) {
  uint8_t *p = b.bytes->data + offset;
  if (little) {
    return ((uint32_t)p[0]) | ((uint32_t)p[1] << 8) | ((uint32_t)p[2] << 16) | ((uint32_t)p[3] << 24);
  }
  return ((uint32_t)p[0] << 24) | ((uint32_t)p[1] << 16) | ((uint32_t)p[2] << 8) | ((uint32_t)p[3]);
}

static uint64_t tya_binary_u64(TyaValue b, int offset, bool little) {
  uint8_t *p = b.bytes->data + offset;
  uint64_t out = 0;
  for (int i = 0; i < 8; i++) {
    int j = little ? 7 - i : i;
    out = (out << 8) | (uint64_t)p[j];
  }
  return out;
}

static void tya_binary_require(TyaValue b, TyaValue offset, int width, const char *name) {
  if (b.kind != TYA_BYTES || b.bytes == NULL) {
    tya_raise(tya_string("binary: data must be bytes"));
    return;
  }
  if (offset.kind != TYA_NUMBER) {
    tya_raise(tya_string("binary: offset must be a number"));
    return;
  }
  int pos = (int)offset.number;
  if (pos < 0 || pos + width > b.bytes->len) {
    tya_raise(tya_string(name));
  }
}

TyaValue tya_binary_read_f32(TyaValue b, TyaValue offset, TyaValue endian) {
  tya_binary_require(b, offset, 4, "binary.read_f32: read past end");
  uint32_t bits = tya_binary_u32(b, (int)offset.number, tya_binary_little(endian));
  float f;
  memcpy(&f, &bits, sizeof(float));
  return tya_number((double)f);
}

TyaValue tya_binary_read_f64(TyaValue b, TyaValue offset, TyaValue endian) {
  tya_binary_require(b, offset, 8, "binary.read_f64: read past end");
  uint64_t bits = tya_binary_u64(b, (int)offset.number, tya_binary_little(endian));
  double f;
  memcpy(&f, &bits, sizeof(double));
  return tya_number(f);
}

static TyaValue tya_binary_write_bits(uint64_t bits, int width, bool little) {
  uint8_t out[8];
  for (int i = 0; i < width; i++) {
    int shift = little ? i * 8 : (width - i - 1) * 8;
    out[i] = (uint8_t)((bits >> shift) & 0xff);
  }
  return tya_bytes_lit((const char *)out, width);
}

TyaValue tya_binary_write_f32(TyaValue value, TyaValue endian) {
  if (value.kind != TYA_NUMBER) {
    tya_raise(tya_string("binary.write_f32: value must be a number"));
    return tya_nil();
  }
  float f = (float)value.number;
  uint32_t bits;
  memcpy(&bits, &f, sizeof(float));
  return tya_binary_write_bits(bits, 4, tya_binary_little(endian));
}

TyaValue tya_binary_write_f64(TyaValue value, TyaValue endian) {
  if (value.kind != TYA_NUMBER) {
    tya_raise(tya_string("binary.write_f64: value must be a number"));
    return tya_nil();
  }
  uint64_t bits;
  memcpy(&bits, &value.number, sizeof(double));
  return tya_binary_write_bits(bits, 8, tya_binary_little(endian));
}

TyaValue tya_stderr_write(TyaValue text) {
  if (text.kind != TYA_STRING || text.string == NULL) {
    tya_raise(tya_string("stderr.write: text must be a string"));
    return tya_nil();
  }
  fputs(text.string, stderr);
  fflush(stderr);
  return tya_nil();
}

TyaValue tya_file_append(TyaValue path, TyaValue text) {
  if (path.kind != TYA_STRING || path.string == NULL) {
    tya_raise(tya_string("file.append: path must be a string"));
    return tya_nil();
  }
  if (text.kind != TYA_STRING || text.string == NULL) {
    tya_raise(tya_string("file.append: text must be a string"));
    return tya_nil();
  }
  FILE *fp = fopen(path.string, "ab");
  if (fp == NULL) {
    tya_raise(tya_string(strerror(errno)));
    return tya_nil();
  }
  fputs(text.string, fp);
  fclose(fp);
  return tya_nil();
}

static bool tya_value_bytes(TyaValue value, const unsigned char **data, size_t *len, const char *op) {
  if (value.kind == TYA_BYTES && value.bytes != NULL) {
    *data = value.bytes->data;
    *len = (size_t)value.bytes->len;
    return true;
  }
  if (value.kind == TYA_STRING && value.string != NULL) {
    *data = (const unsigned char *)value.string;
    *len = strlen(value.string);
    return true;
  }
  char buf[128];
  snprintf(buf, sizeof(buf), "%s: value must be a string or bytes", op);
  tya_raise(tya_string(buf));
  return false;
}

#ifdef TYA_ENABLE_ZLIB
static TyaValue tya_deflate_bytes(TyaValue value, int window_bits, const char *op) {
  const unsigned char *input = NULL;
  size_t input_len = 0;
  if (!tya_value_bytes(value, &input, &input_len, op)) return tya_nil();
  z_stream zs;
  memset(&zs, 0, sizeof(zs));
  if (deflateInit2(&zs, Z_DEFAULT_COMPRESSION, Z_DEFLATED, window_bits, 8, Z_DEFAULT_STRATEGY) != Z_OK) {
    tya_raise(tya_string("compress: deflate init failed"));
    return tya_nil();
  }
  size_t cap = deflateBound(&zs, input_len);
  unsigned char *out = malloc(cap > 0 ? cap : 1);
  if (out == NULL) {
    deflateEnd(&zs);
    tya_raise(tya_string("compress: out of memory"));
    return tya_nil();
  }
  zs.next_in = (Bytef *)input;
  zs.avail_in = (uInt)input_len;
  zs.next_out = out;
  zs.avail_out = (uInt)cap;
  int rc = deflate(&zs, Z_FINISH);
  if (rc != Z_STREAM_END) {
    free(out);
    deflateEnd(&zs);
    tya_raise(tya_string("compress: deflate failed"));
    return tya_nil();
  }
  TyaValue result = tya_bytes_lit((const char *)out, (int)zs.total_out);
  free(out);
  deflateEnd(&zs);
  return result;
}

static TyaValue tya_inflate_bytes(TyaValue value, int window_bits, const char *op) {
  const unsigned char *input = NULL;
  size_t input_len = 0;
  if (!tya_value_bytes(value, &input, &input_len, op)) return tya_nil();
  z_stream zs;
  memset(&zs, 0, sizeof(zs));
  if (inflateInit2(&zs, window_bits) != Z_OK) {
    tya_raise(tya_string("compress: inflate init failed"));
    return tya_nil();
  }
  size_t cap = input_len * 3 + 1024;
  if (cap < 1024) cap = 1024;
  unsigned char *out = malloc(cap);
  if (out == NULL) {
    inflateEnd(&zs);
    tya_raise(tya_string("compress: out of memory"));
    return tya_nil();
  }
  zs.next_in = (Bytef *)input;
  zs.avail_in = (uInt)input_len;
  while (true) {
    zs.next_out = out + zs.total_out;
    zs.avail_out = (uInt)(cap - zs.total_out);
    int rc = inflate(&zs, Z_NO_FLUSH);
    if (rc == Z_STREAM_END) {
      TyaValue result = tya_bytes_lit((const char *)out, (int)zs.total_out);
      free(out);
      inflateEnd(&zs);
      return result;
    }
    if (rc != Z_OK) {
      free(out);
      inflateEnd(&zs);
      tya_raise(tya_string("compress: invalid compressed data"));
      return tya_nil();
    }
    if (zs.total_out == cap) {
      cap *= 2;
      unsigned char *next = realloc(out, cap);
      if (next == NULL) {
        free(out);
        inflateEnd(&zs);
        tya_raise(tya_string("compress: out of memory"));
        return tya_nil();
      }
      out = next;
    }
  }
}

TyaValue tya_compress_gzip(TyaValue value) {
  return tya_deflate_bytes(value, 15 + 16, "compress.gzip");
}

TyaValue tya_compress_gunzip(TyaValue value) {
  return tya_inflate_bytes(value, 15 + 32, "compress.gunzip");
}

TyaValue tya_compress_zlib(TyaValue value) {
  return tya_deflate_bytes(value, 15, "compress.zlib");
}

TyaValue tya_compress_unzlib(TyaValue value) {
  return tya_inflate_bytes(value, 15, "compress.unzlib");
}
#else
static TyaValue tya_zlib_disabled(const char *op) {
  char buf[128];
  snprintf(buf, sizeof(buf), "%s: zlib support is not enabled for this build", op);
  tya_raise(tya_string(buf));
  return tya_nil();
}

TyaValue tya_compress_gzip(TyaValue value) {
  (void)value;
  return tya_zlib_disabled("compress.gzip");
}

TyaValue tya_compress_gunzip(TyaValue value) {
  (void)value;
  return tya_zlib_disabled("compress.gunzip");
}

TyaValue tya_compress_zlib(TyaValue value) {
  (void)value;
  return tya_zlib_disabled("compress.zlib");
}

TyaValue tya_compress_unzlib(TyaValue value) {
  (void)value;
  return tya_zlib_disabled("compress.unzlib");
}
#endif

static TyaValue tya_stream_value(FILE *fp, bool borrowed, bool binary, bool readable, bool writable) {
  TyaResource *r = tya_resource_new(TYA_RES_STREAM);
  r->stream = fp;
  r->stream_borrowed = borrowed;
  r->stream_binary = binary;
  r->stream_readable = readable;
  r->stream_writable = writable;
  r->stream_closed = false;
  return (TyaValue){.kind = TYA_RESOURCE, .resource = r};
}

TyaValue tya_io_stdin(void) {
  return tya_stream_value(stdin, true, false, true, false);
}

TyaValue tya_io_stdout(void) {
  return tya_stream_value(stdout, true, false, false, true);
}

TyaValue tya_io_stderr(void) {
  return tya_stream_value(stderr, true, false, false, true);
}

TyaValue tya_io_open(TyaValue path, TyaValue mode) {
  if (path.kind != TYA_STRING || path.string == NULL) {
    tya_raise(tya_string("io.open: path must be a string"));
    return tya_nil();
  }
  if (mode.kind != TYA_STRING || mode.string == NULL) {
    tya_raise(tya_string("io.open: mode must be a string"));
    return tya_nil();
  }
  const char *m = mode.string;
  bool readable = strchr(m, 'r') != NULL;
  bool writable = strchr(m, 'w') != NULL || strchr(m, 'a') != NULL;
  bool binary = strchr(m, 'b') != NULL;
  if (!readable && !writable) {
    tya_raise(tya_string("io.open: invalid mode"));
    return tya_nil();
  }
  FILE *fp = fopen(path.string, m);
  if (fp == NULL) {
    tya_raise(tya_string(strerror(errno)));
    return tya_nil();
  }
  return tya_stream_value(fp, false, binary, readable, writable);
}

static TyaResource *tya_stream_check(TyaValue stream, const char *op) {
  TyaResource *r = tya_resource_check(stream, TYA_RES_STREAM, op);
  if (r == NULL) return NULL;
  if (r->stream_closed || r->stream == NULL) {
    char buf[128];
    snprintf(buf, sizeof(buf), "%s: stream is closed", op);
    tya_raise(tya_string(buf));
    return NULL;
  }
  return r;
}

static TyaValue tya_string_from_buffer(const char *buf, int len) {
  char *out = malloc((size_t)len + 1);
  if (out == NULL) {
    tya_raise(tya_string("io.read: out of memory"));
    return tya_nil();
  }
  memcpy(out, buf, (size_t)len);
  out[len] = '\0';
  return tya_string(out);
}

TyaValue tya_io_stream_read(TyaValue stream, TyaValue size_v) {
  TyaResource *r = tya_stream_check(stream, "io.read");
  if (r == NULL) return tya_nil();
  if (!r->stream_readable) {
    tya_raise(tya_string("io.read: stream is not readable"));
    return tya_nil();
  }
  if (size_v.kind != TYA_NUMBER) {
    tya_raise(tya_string("io.read: size must be a number"));
    return tya_nil();
  }
  int size = (int)size_v.number;
  if (size < 0) {
    tya_raise(tya_string("io.read: size must be non-negative"));
    return tya_nil();
  }
  char *buf = malloc((size_t)(size > 0 ? size : 1));
  if (buf == NULL) {
    tya_raise(tya_string("io.read: out of memory"));
    return tya_nil();
  }
  size_t got = fread(buf, 1, (size_t)size, r->stream);
  TyaValue out = r->stream_binary ? tya_bytes_lit(buf, (int)got) : tya_string_from_buffer(buf, (int)got);
  free(buf);
  return out;
}

TyaValue tya_io_stream_read_line(TyaValue stream) {
  TyaResource *r = tya_stream_check(stream, "io.read_line");
  if (r == NULL) return tya_nil();
  if (!r->stream_readable) {
    tya_raise(tya_string("io.read_line: stream is not readable"));
    return tya_nil();
  }
  if (feof(r->stream)) return tya_nil();
  size_t cap = 128;
  size_t len = 0;
  char *buf = malloc(cap);
  if (buf == NULL) {
    tya_raise(tya_string("io.read_line: out of memory"));
    return tya_nil();
  }
  int ch;
  while ((ch = fgetc(r->stream)) != EOF) {
    if (len + 1 >= cap) {
      cap *= 2;
      char *next = realloc(buf, cap);
      if (next == NULL) {
        free(buf);
        tya_raise(tya_string("io.read_line: out of memory"));
        return tya_nil();
      }
      buf = next;
    }
    buf[len++] = (char)ch;
    if (ch == '\n') break;
  }
  if (len == 0 && ch == EOF) {
    free(buf);
    return tya_nil();
  }
  TyaValue out = r->stream_binary ? tya_bytes_lit(buf, (int)len) : tya_string_from_buffer(buf, (int)len);
  free(buf);
  return out;
}

TyaValue tya_io_stream_eof(TyaValue stream) {
  TyaResource *r = tya_stream_check(stream, "io.eof?");
  if (r == NULL) return tya_bool(true);
  return tya_bool(feof(r->stream));
}

TyaValue tya_io_stream_write(TyaValue stream, TyaValue value) {
  TyaResource *r = tya_stream_check(stream, "io.write");
  if (r == NULL) return tya_nil();
  if (!r->stream_writable) {
    tya_raise(tya_string("io.write: stream is not writable"));
    return tya_nil();
  }
  if (value.kind == TYA_BYTES && value.bytes != NULL) {
    if (value.bytes->len > 0) fwrite(value.bytes->data, 1, (size_t)value.bytes->len, r->stream);
    return tya_number(value.bytes->len);
  }
  TyaValue s = tya_to_string(value);
  if (s.string == NULL) return tya_number(0);
  fputs(s.string, r->stream);
  return tya_number(strlen(s.string));
}

TyaValue tya_io_stream_flush(TyaValue stream) {
  TyaResource *r = tya_stream_check(stream, "io.flush");
  if (r == NULL) return tya_nil();
  fflush(r->stream);
  return tya_nil();
}

TyaValue tya_io_stream_close(TyaValue stream) {
  TyaResource *r = tya_resource_check(stream, TYA_RES_STREAM, "io.close");
  if (r == NULL) return tya_nil();
  if (r->stream_closed) return tya_nil();
  if (!r->stream_borrowed && r->stream != NULL) fclose(r->stream);
  r->stream_closed = true;
  r->stream = NULL;
  return tya_nil();
}

