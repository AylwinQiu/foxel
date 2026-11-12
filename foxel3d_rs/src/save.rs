//! 存档：一个文件夹，里面两个文件。
//!
//! - world.json：版本、种子、方块名字表、场，还有Lua给的meta（玩家、物品栏、mod数据……），人可以直接看和改
//! - chunks.bin：gzip压缩的二进制，只有改动过的chunk，没改过的下次重新生成（同一个种子生成的一样）
//!
//! 方块按名字保存（palette），所以mod增删了方块、注册顺序变了都能读；找不到的方块变成空气。
//! 每个文件都是先写到临时文件再改名，写到一半出错不会弄坏原来的存档。

use std::collections::HashMap;
use std::fs;
use std::io::{self, Read, Write};
use std::path::Path;

use flate2::Compression;
use flate2::read::GzDecoder;
use flate2::write::GzEncoder;
use serde::{Deserialize, Serialize};

use crate::world::CubeId;
use crate::world::encode::{get_uvarint, put_uvarint};

pub const SAVE_VERSION: u32 = 1;
const CHUNKS_MAGIC: &[u8; 4] = b"FXC1";

pub struct SaveData {
    pub seed: i64,
    /// 存档里BlockId i对应的方块名字
    pub palette: Vec<String>,
    pub chunks: HashMap<CubeId, Vec<u8>>,
    pub fields: HashMap<String, HashMap<CubeId, String>>,
    /// Lua的数据，原样保存
    pub meta: serde_json::Value,
}

/// 一个场在world.json里的样子：[[x, y, z, depth, 值], ...]
type FieldList = Vec<(i64, i64, i64, i32, String)>;

#[derive(Serialize, Deserialize)]
struct WorldJson {
    version: u32,
    seed: i64,
    palette: Vec<String>,
    fields: HashMap<String, FieldList>,
    meta: serde_json::Value,
}

fn other(msg: String) -> io::Error {
    io::Error::new(io::ErrorKind::InvalidData, msg)
}

fn write_atomic(path: &Path, data: &[u8]) -> io::Result<()> {
    let tmp = path.with_extension("tmp");
    let result = fs::File::create(&tmp).and_then(|mut f| {
        f.write_all(data)?;
        f.sync_all()
    });
    if let Err(e) = result {
        let _ = fs::remove_file(&tmp);
        return Err(e);
    }
    fs::rename(&tmp, path)
}

fn zigzag(v: i64) -> u64 {
    ((v << 1) ^ (v >> 63)) as u64
}

fn unzigzag(v: u64) -> i64 {
    (v >> 1) as i64 ^ -((v & 1) as i64)
}

pub fn write(dir: &Path, s: &SaveData) -> io::Result<()> {
    fs::create_dir_all(dir)?;

    // chunks.bin: 魔数，chunk个数，然后每个chunk：x, y, z（zigzag uvarint），数据长度，数据
    let mut buf = CHUNKS_MAGIC.to_vec();
    put_uvarint(&mut buf, s.chunks.len() as u64);
    let mut chunks: Vec<_> = s.chunks.iter().collect();
    chunks.sort_by_key(|(pos, _)| (pos.x, pos.y, pos.z));
    for (pos, data) in chunks {
        for v in [pos.x, pos.y, pos.z] {
            put_uvarint(&mut buf, zigzag(v));
        }
        put_uvarint(&mut buf, data.len() as u64);
        buf.extend_from_slice(data);
    }
    let mut z = GzEncoder::new(Vec::new(), Compression::default());
    z.write_all(&buf)?;
    write_atomic(&dir.join("chunks.bin"), &z.finish()?)?;

    let fields = s
        .fields
        .iter()
        .map(|(name, f)| {
            let mut list: Vec<_> = f.iter().map(|(id, v)| (id.x, id.y, id.z, id.depth, v.clone())).collect();
            list.sort();
            (name.clone(), list)
        })
        .collect();
    let json =
        WorldJson { version: SAVE_VERSION, seed: s.seed, palette: s.palette.clone(), fields, meta: s.meta.clone() };
    let text = serde_json::to_string_pretty(&json).map_err(|e| other(e.to_string()))?;
    write_atomic(&dir.join("world.json"), text.as_bytes())
}

/// 没有存档时返回None
pub fn read(dir: &Path) -> io::Result<Option<SaveData>> {
    let text = match fs::read_to_string(dir.join("world.json")) {
        Ok(t) => t,
        Err(e) if e.kind() == io::ErrorKind::NotFound => return Ok(None),
        Err(e) => return Err(e),
    };
    let json: WorldJson = serde_json::from_str(&text).map_err(|e| other(format!("world.json格式不对：{e}")))?;
    if json.version != SAVE_VERSION {
        return Err(other(format!("存档的版本是{}，现在只能读版本{SAVE_VERSION}", json.version)));
    }

    let mut chunks = HashMap::new();
    match fs::File::open(dir.join("chunks.bin")) {
        Ok(f) => {
            let mut buf = Vec::new();
            GzDecoder::new(f).read_to_end(&mut buf)?;
            let bad = || other("chunks.bin格式不对".into());
            let mut rest = buf.strip_prefix(CHUNKS_MAGIC.as_slice()).ok_or_else(bad)?;
            let next = |rest: &mut &[u8]| -> io::Result<u64> {
                let (v, k) = get_uvarint(rest).ok_or_else(bad)?;
                *rest = &rest[k..];
                Ok(v)
            };
            let n = next(&mut rest)?;
            for _ in 0..n {
                let x = unzigzag(next(&mut rest)?);
                let y = unzigzag(next(&mut rest)?);
                let z = unzigzag(next(&mut rest)?);
                let len = next(&mut rest)? as usize;
                if rest.len() < len {
                    return Err(bad());
                }
                chunks.insert(CubeId::new(x, y, z, crate::world::CHUNK_DEPTH), rest[..len].to_vec());
                rest = &rest[len..];
            }
        }
        Err(e) if e.kind() == io::ErrorKind::NotFound => {}
        Err(e) => return Err(e),
    }

    let fields = json
        .fields
        .into_iter()
        .map(|(name, list)| (name, list.into_iter().map(|(x, y, z, d, v)| (CubeId::new(x, y, z, d), v)).collect()))
        .collect();
    Ok(Some(SaveData { seed: json.seed, palette: json.palette, chunks, fields, meta: json.meta }))
}
