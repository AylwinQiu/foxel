//! chunk的编码，用于存档。
//!
//! 先序遍历八叉树，每个cube写一个uvarint：0表示分成8个子cube（后面跟着8个子cube的编码），
//! 否则是叶子，值是block+1。没有改动过的chunk不保存，加载时重新生成。

use std::collections::HashMap;

use super::{BlockId, CHUNK_DEPTH, Cube, CubeId, Depth, MIN_DEPTH, World};

#[derive(Debug)]
pub struct BadChunk;

impl std::fmt::Display for BadChunk {
    fn fmt(&self, f: &mut std::fmt::Formatter) -> std::fmt::Result {
        write!(f, "chunk数据格式不对")
    }
}

pub fn put_uvarint(buf: &mut Vec<u8>, mut v: u64) {
    while v >= 0x80 {
        buf.push(v as u8 | 0x80);
        v >>= 7;
    }
    buf.push(v as u8);
}

/// 读一个uvarint，返回值和读了几个字节
pub fn get_uvarint(data: &[u8]) -> Option<(u64, usize)> {
    let mut v = 0u64;
    for (i, &b) in data.iter().enumerate().take(10) {
        v |= ((b & 0x7f) as u64) << (7 * i);
        if b < 0x80 {
            return Some((v, i + 1));
        }
    }
    None
}

pub fn encode_chunk(n: &Cube) -> Vec<u8> {
    let mut buf = Vec::new();
    encode_cube(&mut buf, n);
    buf
}

fn encode_cube(buf: &mut Vec<u8>, n: &Cube) {
    match &n.children {
        None => put_uvarint(buf, n.block as u64 + 1),
        Some(children) => {
            buf.push(0);
            for c in children.iter() {
                encode_cube(buf, c);
            }
        }
    }
}

pub fn decode_chunk(data: &[u8]) -> Result<Cube, BadChunk> {
    let mut rest = data;
    let n = decode_cube(&mut rest, CHUNK_DEPTH)?;
    if rest.is_empty() { Ok(n) } else { Err(BadChunk) }
}

fn decode_cube(data: &mut &[u8], depth: Depth) -> Result<Cube, BadChunk> {
    let (v, k) = get_uvarint(data).ok_or(BadChunk)?;
    *data = &data[k..];
    if v > 1 << 16 {
        return Err(BadChunk);
    }
    if v != 0 {
        return Ok(Cube::leaf((v - 1) as BlockId));
    }
    if depth <= MIN_DEPTH {
        return Err(BadChunk);
    }
    let mut children: [Cube; 8] = Default::default();
    for c in children.iter_mut() {
        *c = decode_cube(data, depth - 1)?;
    }
    Ok(Cube { block: 0, children: Some(Box::new(children)) })
}

impl World {
    /// 所有改动过的chunk的编码，包括已加载的和卸载了的（存档用）
    pub fn saved_chunks(&self) -> HashMap<CubeId, Vec<u8>> {
        let mut out = self.saved.clone();
        for (pos, ch) in &self.chunks {
            if ch.modified {
                out.insert(*pos, encode_chunk(&ch.node));
            }
        }
        out
    }

    /// 换成存档里的chunk。remap把存档里的BlockId换成现在的BlockId（注册顺序可能变了），
    /// 数据坏了的chunk会被丢掉（之后重新生成），返回丢掉了几个
    pub fn set_saved_chunks(&mut self, chunks: HashMap<CubeId, Vec<u8>>, remap: &dyn Fn(BlockId) -> BlockId) -> usize {
        self.saved.clear();
        let mut dropped = 0;
        for (pos, data) in chunks {
            match decode_chunk(&data) {
                Ok(mut n) if pos.depth == CHUNK_DEPTH => {
                    remap_cube(&mut n, remap);
                    self.saved.insert(pos, encode_chunk(&n));
                }
                _ => dropped += 1,
            }
        }
        dropped
    }
}

fn remap_cube(n: &mut Cube, remap: &dyn Fn(BlockId) -> BlockId) {
    match &mut n.children {
        None => n.block = remap(n.block),
        Some(children) => {
            for c in children.iter_mut() {
                remap_cube(c, remap);
            }
            n.try_merge();
        }
    }
}
