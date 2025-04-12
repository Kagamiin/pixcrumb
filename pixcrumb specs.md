# Kagamiin's bitmap compression format for the NES

- Bitplanes are compressed separately
  - They can be preprocessed by XOR'ing them together to exploit redundancy between them

- Bitplanes are split into "fragments"
  - 2x2 pixel blocks
  - Each fragment is represented by 4 bits
  - Group pixels together in order to form a symbol to apply compression over

- Bitplanes are delta-encoded
  - Delta encoding is applied vertically
  - The premise is that the decoding can be streamed, reducing memory usage

- RLE/LZ is performed horizontally
  - The combination of vertical delta encoding and horizontal compression provides redundancy encoding in both dimensions

## Format 1 - simple

- Contains data stream and run-length stream
  - Data stream encodes in nibbles
  - Run-length stream is a bitstream

- Switches between 2 modes

- Mode 1 - literal mode
  - Read nibble from data stream
  - If nibble is zero, switch to Mode 2
  - Otherwise, add it to the output and continue

- Mode 2 - zero-run mode
  - Read n = order-2 Exp-Golomb encoded number from run-length stream
  - Add n + 1 zeroes to the output
  - Switch to Mode 1

- Always starts in Mode 1

- Format:
  - u16: 16-bit offset of data stream from beginning of file
  - u8: Height in fragments (pixels / 2)
  - u8: Width in tiles (1-8)
  - byte[]: Run-length stream (padded to the next byte with zeroes)
  - byte[]: Data stream (padded to the next byte with zeroes)
  
- Stops decoding when it reaches the end of the image

## Format 2 - with single literal

- Contains data stream and command stream
  - Data stream encodes in nibbles
  - Command stream is a bitstream

- Commands are encoded as a decision tree in decoder code
- Read 2-bit command from command stream and determine what to do

- 00: Add a single zero to the output

- 01: Zero-RLE
  - Read n = order-1 Exp-Golomb encoded number from command stream
  - Add n + 3 zeroes to the output

- 10: Single literal
  - Read nibble from data stream
  - Add it to the output

- 11: Literal mode - loop:
  - Read nibble from data stream
  - Add it to the output
  - If nibble is zero, stop
  - Otherwise, continue

- Order-1 Exp-Golomb coding is used because it produces numbers with even length, maintaining the bit alignment and perhaps facilitating decoding of the command stream

- Format:
  - u16: 16-bit offset of data stream from beginning of file
  - u8: Height in fragments (pixels / 2)
  - u8: Width in tiles (1-8)
  - byte[]: Command stream (padded to the next byte with zeroes)
  - byte[]: Data stream (padded to the next byte with zeroes)

- Stops decoding when it reaches the end of the image

## Format 3 - with literal RLE

- Contains data stream and command stream
  - Data stream encodes in nibbles
  - Command stream is a bitstream

- Commands are encoded as a decision tree in decoder code
- Read 2-bit command from command stream and determine what to do

- 00: Add a single zero to the output

- 01: Zero-RLE
  - Read n = order-1 Exp-Golomb encoded number from command stream
  - Add n + 3 zeroes to the output

- 10: Literal RLE
  - Read n = order-1 Exp-Golomb encoded number from command stream
  - Read d = nibble from data stream
  - Add n + 3 fragments of value d to the output

- 11: Literal mode - loop:
  - Read nibble from data stream
  - Add it to the output
  - If nibble is zero, stop
  - Otherwise, continue

## Format 4 - with single literal and literal RLE

- Contains data stream and command stream
  - Data stream encodes in nibbles
  - Command stream is a bitstream

- Commands are encoded as a decision tree in decoder code
- Read variable-length command from command stream and determine what to do

- 00: Add a single zero to the output

- 01: Single literal
  - Read nibble from data stream
  - Add it to the output

- 100: Zero-RLE
  - Read n = order-1 Exp-Golomb encoded number from command stream
  - Add n + 3 zeroes to the output

- 101: Literal RLE
  - Read n = order-1 Exp-Golomb encoded number from command stream
  - Read d = nibble from data stream
  - Add n + 3 fragments of value d to the output

- 11: Literal mode - loop:
  - Read nibble from data stream
  - If nibble is zero, stop
  - Otherwise, add it to the output and continue

## Format 5 - with LZ

- Contains data stream and command stream
  - Data stream encodes in nibbles
  - Command stream is a bitstream

- Commands are encoded as a decision tree in decoder code
- Read 2-bit command from command stream and determine what to do

- 00: Add a single zero to the output

- 01: Zero-RLE
  - Read n = order-1 Exp-Golomb encoded number from command stream
  - Add n + 3 zeroes to the output

- 10: LZ
  - Read offset = order-1 Exp-Golomb encoded number from command stream
  - Read len = order-1 Exp-Golomb encoded number from command stream
  - Loop (len + 3) times:
    - Read nibble from (offset + 1) fragments behind the output position
    - Add it to the output and continue

- 11: Literal mode - loop:
  - Read nibble from data stream
  - If nibble is zero, stop
  - Otherwise, add it to the output and continue

## Format 5 - simplified, with LZ

- Contains data stream and run-length stream
  - Data stream encodes in nibbles
  - Run-length stream is a bitstream

- Switches between 2 modes

- Mode 1 - literal mode
  - Read nibble from data stream
  - If nibble is zero, switch to Mode 2
  - Otherwise, add it to the output and continue

- Mode 2 - LZ mode
  - Read len = order-2 Exp-Golomb encoded number from command stream
  - If len == 0, switch to Mode 1
  - Otherwise:
    - Read offset = order-2 Exp-Golomb encoded number from command stream
    - Loop (len) times:
      - Read nibble from (offset + 1) fragments behind the output position
      - Add it to the output and continue

- Always starts in Mode 1


## Format 6 - with single literal and LZ

- Contains data stream and command stream
  - Data stream encodes in nibbles
  - Command stream is a bitstream

- Commands are encoded as a decision tree in decoder code
- Read variable-length command from command stream and determine what to do

- 00: Add a single zero to the output

- 01: Single literal
  - Read nibble from data stream
  - Add it to the output

- 100: Zero-RLE
  - Read n = order-1 Exp-Golomb encoded number from command stream
  - Add n + 3 zeroes to the output

- 101: LZ
  - Read offset = order-1 Exp-Golomb encoded number from command stream
  - Read len = order-1 Exp-Golomb encoded number from command stream
  - Loop (len + 3) times:
    - Read nibble from (offset + 1) fragments behind the output position
    - Add it to the output and continue

- 11: Literal mode - loop:
  - Read nibble from data stream
  - If nibble is zero, stop
  - Otherwise, add it to the output and continue

## Format 7 - with single literal, literal RLE and complex LZ

- Contains data stream and command stream
  - Data stream encodes in nibbles
  - Command stream is a bitstream

- Commands are encoded as a decision tree in decoder code
- Read variable-length command from command stream and determine what to do

- 00: Add a single zero to the output

- 01: Single literal
  - Read nibble from data stream
  - Add it to the output

- 100: Zero-RLE
  - Read n = order-1 Exp-Golomb encoded number from command stream
  - Add n + 3 zeroes to the output

- 1010: Literal RLE
  - Read n = order-1 Exp-Golomb encoded number from command stream
  - Read d = nibble from data stream
  - Add n + 3 fragments of value d to the output

- 1011: Short/fast LZ
  - Read offset = 2 nibbles from data stream (little-endian)
  - Read len = nibble from data stream
  - Loop (len + 4) times:
    - Read nibble from (offset + 1) fragments behind the output position
    - Add it to the output and continue

- 11000: LZ
  - Read offset = order-1 Exp-Golomb encoded number from command stream
  - Read len = order-1 Exp-Golomb encoded number from command stream
  - Loop (len + 3) times:
    - Read nibble from (offset + 1) fragments behind the output position
    - Add it to the output and continue

- 11001: Reverse LZ
  - Read offset = order-1 Exp-Golomb encoded number from command stream
  - Read len = order-1 Exp-Golomb encoded number from command stream
  - Loop (len + 3) times:
    - Read nibble from (offset + 1) fragments behind the output position
    - Add it to the output
    - Increment offset by 2 and continue
    - This will essentially read the output buffer backwards

- 11010: LZ with data inversion
  - Read offset = order-1 Exp-Golomb encoded number from command stream
  - Read len = order-1 Exp-Golomb encoded number from command stream
  - Loop (len + 3) times:
    - Read nibble from (offset + 1) fragments behind the output position
    - Invert the bits of the nibble
    - Add it to the output and continue

- 11011: Reverse LZ with data inversion
  - Read offset = order-1 Exp-Golomb encoded number from command stream
  - Read len = order-1 Exp-Golomb encoded number from command stream
  - Loop (len + 3) times:
    - Read nibble from (offset + 1) fragments behind the output position
    - Invert the bits of the nibble
    - Add it to the output
    - Increment offset by 2 and continue

- 111: Literal mode - loop:
  - Read nibble from data stream
  - If nibble is zero, stop
  - Otherwise, add it to the output and continue
