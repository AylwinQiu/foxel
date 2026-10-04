vim.o.updatetime = 300

vim.api.nvim_create_autocmd("CursorHold", {
  pattern = "*.rs",
  callback = function()
    vim.lsp.buf.hover()
  end,
})

vim.api.nvim_create_autocmd("FileType", {
  pattern = "rust",
  callback = function(args)
    vim.lsp.start({
      name = "rust-analyzer",
      cmd = { "rust-analyzer" },
      root_dir = vim.fs.root(args.buf, { "Cargo.toml", ".git" }),
      settings = {
        ["rust-analyzer"] = {
          check = {
            command = "clippy",
          },
        },
      },
    })
  end,
})
