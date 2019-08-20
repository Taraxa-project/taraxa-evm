#ifndef TARAXA_EVM_TARAXA_CGO_DB_INDEX_H
#define TARAXA_EVM_TARAXA_CGO_DB_INDEX_H

#include <stdbool.h>
#include <stdint.h>

#define ID(name) taraxa_cgo_ethdb_##name

typedef char ID(SliceType);
typedef int ID(SliceSize);

typedef struct {
  ID(SliceType) * offset;
  ID(SliceSize) size;
} ID(Slice);

inline ID(Slice) ID(Slice_New)(ID(SliceType) * offset, ID(SliceSize) size) {
  return (ID(Slice)){offset, size};
}

typedef ID(Slice) ID(Error);
typedef ID(Slice) ID(Key);
typedef ID(Slice) ID(Value);

typedef struct {
  void *self;
  void (*Free)(void *);
  ID(Error) (*Put)(void *, ID(Key), ID(Value));
  ID(Error) (*Delete)(void *, ID(Key));
  ID(Error) (*Write)(void *);
  void (*Reset)(void *);
} ID(Batch);

inline void ID(Batch_Free)(ID(Batch) * self) { self->Free(self->self); }

inline ID(Error) ID(Batch_Put)(ID(Batch) * self, ID(Key) key, ID(Value) value) {
  return self->Put(self->self, key, value);
}

inline ID(Error) ID(Batch_Delete)(ID(Batch) * self, ID(Key) key) {
  return self->Delete(self->self, key);
}

inline ID(Error) ID(Batch_Write)(ID(Batch) * self) {
  return self->Write(self->self);
}

inline void ID(Batch_Reset)(ID(Batch) * self) { self->Reset(self->self); }

typedef struct {
  bool ret;
  ID(Error) err;
} ID(BoolAndErr);

typedef struct {
  ID(Value) ret;
  ID(Error) err;
} ID(ValueAndErr);

typedef struct {
  void *self;
  void (*Free)(void *);
  ID(Error) (*Put)(void *, ID(Key), ID(Value));
  ID(Error) (*Delete)(void *, ID(Key));
  ID(ValueAndErr) (*Get)(void *, ID(Key));
  ID(BoolAndErr) (*Has)(void *, ID(Key));
  void (*Close)(void *);
  ID(Batch) * (*NewBatch)(void *);
} ID(Database);

inline void ID(Database_Free)(ID(Database) * self) { self->Free(self->self); }

inline ID(Error)
    ID(Database_Put)(ID(Database) * self, ID(Key) key, ID(Value) value) {
  return self->Put(self->self, key, value);
}

inline ID(Error) ID(Database_Delete)(ID(Database) * self, ID(Key) key) {
  return self->Delete(self->self, key);
}

inline ID(ValueAndErr) ID(Database_Get)(ID(Database) * self, ID(Key) key) {
  return self->Get(self->self, key);
}

inline ID(BoolAndErr) ID(Database_Has)(ID(Database) * self, ID(Key) key) {
  return self->Has(self->self, key);
}

inline void ID(Database_Close)(ID(Database) * self) { self->Close(self->self); }

inline ID(Batch) * ID(Database_NewBatch)(ID(Database) * self) {
  return self->NewBatch(self->self);
}

#undef ID

#endif // TARAXA_EVM_TARAXA_CGO_DB_INDEX_H