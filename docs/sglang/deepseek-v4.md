> ## Documentation Index
> Fetch the complete documentation index at: https://docs.sglang.io/llms.txt
> Use this file to discover all available pages before exploring further.

# DeepSeek-V4

export const DeepSeekV4Deployment = () => {
  const options = {
    hardware: {
      name: "hardware",
      title: "Hardware Platform",
      items: [{
        id: "b200",
        label: "B200",
        default: true
      }, {
        id: "b300",
        label: "B300",
        default: false
      }, {
        id: "gb200",
        label: "GB200",
        default: false
      }, {
        id: "gb300",
        label: "GB300",
        default: false
      }, {
        id: "h200",
        label: "H200",
        default: false
      }, {
        id: "h100",
        label: "H100",
        default: false
      }, {
        id: "rtx6000",
        label: "RTX PRO 6000",
        default: false
      }]
    },
    modelSize: {
      name: "modelSize",
      title: "Model Variant",
      items: [{
        id: "small",
        label: "Flash",
        default: true,
        subtitle: "285B"
      }, {
        id: "big",
        label: "Pro",
        default: false,
        subtitle: "1.6T"
      }]
    },
    quantization: {
      name: "quantization",
      title: "Quantization",
      items: [{
        id: "fp4",
        label: "FP4",
        default: true
      }, {
        id: "fp8",
        label: "FP8",
        default: false,
        subtitle: "H100/H200 only"
      }]
    },
    recipe: {
      name: "recipe",
      title: "Recipe",
      items: [{
        id: "low-latency",
        label: "Low-Latency",
        default: true
      }, {
        id: "balanced",
        label: "Balanced",
        default: false
      }, {
        id: "max-throughput",
        label: "Max-Throughput",
        default: false
      }, {
        id: "cp",
        label: "Context-Parallel",
        default: false
      }, {
        id: "pd-disagg",
        label: "PD-Disagg",
        default: false
      }]
    },
    reasoningParser: {
      name: "reasoningParser",
      title: "Reasoning Parser",
      items: [{
        id: "disabled",
        label: "Disabled",
        default: true
      }, {
        id: "enabled",
        label: "Enabled",
        default: false,
        subtitle: "deepseek-v4"
      }]
    },
    toolcall: {
      name: "toolcall",
      title: "Tool Call Parser",
      items: [{
        id: "disabled",
        label: "Disabled",
        default: true
      }, {
        id: "enabled",
        label: "Enabled",
        default: false,
        subtitle: "deepseekv4"
      }]
    },
    hicache: {
      name: "hicache",
      title: "HiCache",
      items: [{
        id: "disabled",
        label: "Disabled",
        default: true
      }, {
        id: "l2",
        label: "L2",
        default: false,
        subtitle: "GPU+CPU"
      }]
    },
    megamoe: {
      name: "megamoe",
      title: "MegaMoE",
      items: [{
        id: "disabled",
        label: "Disabled",
        default: true
      }, {
        id: "w4a8",
        label: "W4A8",
        default: false
      }, {
        id: "w4a4",
        label: "W4A4",
        default: false,
        subtitle: "FP4 acts"
      }]
    }
  };
  const FP8_SUPPORTED_HARDWARE = new Set(["h100", "h200"]);
  const effHw = (hardware, quantization) => {
    if (hardware === "h200") return quantization === "fp8" ? "h200" : "h200-fp4";
    if (hardware === "h100") return quantization === "fp8" ? "h100-fp8" : "h100";
    return hardware;
  };
  const MARLIN_UNSUPPORTED_RECIPES = new Set(["cp", "pd-disagg"]);
  const MARLIN_EFFHW = new Set(["h200-fp4", "h100"]);
  const MARLIN_LABEL = {
    "h200-fp4": "H200 (FP4)",
    h100: "H100 (FP4)"
  };
  const MEGAMOE_UNSUPPORTED_RECIPES = new Set(["low-latency", "balanced", "cp", "pd-disagg"]);
  const MEGAMOE_UNSUPPORTED_HARDWARE = new Set(["h100", "h200", "rtx6000"]);
  const isMegamoeUnsupported = vals => MEGAMOE_UNSUPPORTED_HARDWARE.has(vals.hardware) || MEGAMOE_UNSUPPORTED_RECIPES.has(vals.recipe);
  const HICACHE_UNSUPPORTED_RECIPES = new Set(["pd-disagg"]);
  const HICACHE_UNSUPPORTED_HARDWARE = new Set(["rtx6000"]);
  const isHicacheUnsupported = vals => HICACHE_UNSUPPORTED_HARDWARE.has(vals.hardware) || HICACHE_UNSUPPORTED_RECIPES.has(vals.recipe);
  const isProDisabledFp8H100 = vals => vals.hardware === "h100" && vals.quantization === "fp8";
  const resolveItems = (option, vals) => {
    const eff = vals ? effHw(vals.hardware, vals.quantization) : null;
    if (option.name === "recipe" && eff && MARLIN_EFFHW.has(eff)) {
      return option.items.map(it => MARLIN_UNSUPPORTED_RECIPES.has(it.id) ? {
        ...it,
        disabled: true,
        disabledReason: `Not supported on ${MARLIN_LABEL[eff]}`
      } : it);
    }
    if (option.name === "recipe" && eff === "h100-fp8") {
      return option.items.map(it => MARLIN_UNSUPPORTED_RECIPES.has(it.id) ? {
        ...it,
        disabled: true,
        disabledReason: "Not supported on H100 (SGLang FP8)"
      } : it);
    }
    if (option.name === "megamoe" && vals && isMegamoeUnsupported(vals)) {
      const reason = MEGAMOE_UNSUPPORTED_HARDWARE.has(vals.hardware) ? "MegaMoE is only supported on Blackwell" : vals.recipe === "pd-disagg" ? "MegaMoE is not yet wired into the PD-Disagg cookbook command" : "MegaMoE is not supported on this recipe";
      return option.items.map(it => it.id === "disabled" ? it : {
        ...it,
        disabled: true,
        disabledReason: reason
      });
    }
    if (option.name === "hicache" && vals && isHicacheUnsupported(vals)) {
      const reason = HICACHE_UNSUPPORTED_HARDWARE.has(vals.hardware) ? "HiCache is not supported on RTX PRO 6000" : "HiCache is not yet wired into the PD-Disagg cookbook command";
      return option.items.map(it => it.id === "disabled" ? it : {
        ...it,
        disabled: true,
        disabledReason: reason
      });
    }
    if (option.name === "quantization" && vals && !FP8_SUPPORTED_HARDWARE.has(vals.hardware)) {
      return option.items.map(it => it.id === "fp8" ? {
        ...it,
        disabled: true,
        disabledReason: "SGLang FP8 is only available on H100 / H200"
      } : it);
    }
    if (option.name === "modelSize" && vals && vals.hardware === "rtx6000") {
      return option.items.map(it => it.id === "big" ? {
        ...it,
        disabled: true,
        disabledReason: "V4-Pro does not fit on RTX PRO 6000 (8× 96 GB)"
      } : it);
    }
    if (option.name === "recipe" && vals && vals.hardware === "rtx6000") {
      const rtx6000Unsupported = new Set(["balanced", "max-throughput", "cp", "pd-disagg"]);
      return option.items.map(it => rtx6000Unsupported.has(it.id) ? {
        ...it,
        disabled: true,
        disabledReason: "RTX PRO 6000 supports low-latency (TP-only) recipe"
      } : it);
    }
    if (option.name === "modelSize" && vals && isProDisabledFp8H100(vals)) {
      return option.items.map(it => it.id === "big" ? {
        ...it,
        disabled: true,
        disabledReason: "H100 SGLang FP8 only ships a Flash variant"
      } : it);
    }
    return option.items;
  };
  const getInitialState = () => {
    const initialState = {};
    for (const [key, option] of Object.entries(options)) {
      const items = resolveItems(option);
      const def = items.find(i => i.default && !i.disabled) || items.find(i => !i.disabled) || items[0];
      initialState[key] = def.id;
    }
    return initialState;
  };
  const [values, setValues] = useState(getInitialState);
  const [isDark, setIsDark] = useState(false);
  useEffect(() => {
    const checkDarkMode = () => {
      const html = document.documentElement;
      const isDarkMode = html.classList.contains("dark") || html.getAttribute("data-theme") === "dark" || html.style.colorScheme === "dark";
      setIsDark(isDarkMode);
    };
    checkDarkMode();
    const observer = new MutationObserver(checkDarkMode);
    observer.observe(document.documentElement, {
      attributes: true,
      attributeFilter: ["class", "data-theme", "style"]
    });
    return () => observer.disconnect();
  }, []);
  const handleRadioChange = (optionName, value) => {
    setValues(prev => {
      const next = {
        ...prev,
        [optionName]: value
      };
      if (optionName === "hardware" && next.quantization === "fp8" && !FP8_SUPPORTED_HARDWARE.has(value)) {
        next.quantization = "fp4";
      }
      if ((optionName === "hardware" || optionName === "quantization") && isProDisabledFp8H100(next) && next.modelSize === "big") {
        next.modelSize = "small";
      }
      const nextEff = effHw(next.hardware, next.quantization);
      if ((optionName === "hardware" || optionName === "quantization") && (MARLIN_EFFHW.has(nextEff) || nextEff === "h100-fp8") && MARLIN_UNSUPPORTED_RECIPES.has(next.recipe)) {
        next.recipe = "low-latency";
      }
      if ((optionName === "hardware" || optionName === "recipe") && next.megamoe !== "disabled" && isMegamoeUnsupported(next)) {
        next.megamoe = "disabled";
      }
      if ((optionName === "hardware" || optionName === "recipe") && next.hicache !== "disabled" && isHicacheUnsupported(next)) {
        next.hicache = "disabled";
      }
      if ((optionName === "recipe" || optionName === "hardware") && next.recipe === "max-throughput" && next.megamoe === "disabled" && !isMegamoeUnsupported(next)) {
        next.megamoe = "w4a8";
      }
      return next;
    });
  };
  const HW_SIZE_SPEC = {
    "b200|small": {
      slug: "deepseek-ai/DeepSeek-V4-Flash",
      tp: 4,
      multinode: false
    },
    "b200|big": {
      slug: "deepseek-ai/DeepSeek-V4-Pro",
      tp: 8,
      multinode: false
    },
    "gb300|small": {
      slug: "deepseek-ai/DeepSeek-V4-Flash",
      tp: 4,
      multinode: false
    },
    "gb300|big": {
      slug: "deepseek-ai/DeepSeek-V4-Pro",
      tp: 4,
      multinode: false
    },
    "gb200|small": {
      slug: "deepseek-ai/DeepSeek-V4-Flash",
      tp: 4,
      multinode: false
    },
    "gb200|big": {
      slug: "deepseek-ai/DeepSeek-V4-Pro",
      tp: 8,
      multinode: true,
      nnodes: 2
    },
    "h200|small": {
      slug: "sgl-project/DeepSeek-V4-Flash-FP8",
      tp: 4,
      multinode: false
    },
    "h200|big": {
      slug: "sgl-project/DeepSeek-V4-Pro-FP8",
      tp: 16,
      multinode: true,
      nnodes: 2
    },
    "h200-fp4|small": {
      slug: "deepseek-ai/DeepSeek-V4-Flash",
      tp: 4,
      multinode: false
    },
    "h200-fp4|big": {
      slug: "deepseek-ai/DeepSeek-V4-Pro",
      tp: 8,
      multinode: false
    },
    "h100|small": {
      slug: "deepseek-ai/DeepSeek-V4-Flash",
      tp: 8,
      multinode: false
    },
    "h100|big": {
      slug: "deepseek-ai/DeepSeek-V4-Pro",
      tp: 16,
      multinode: true,
      nnodes: 2
    },
    "h100-fp8|small": {
      slug: "sgl-project/DeepSeek-V4-Flash-FP8",
      tp: 8,
      multinode: false
    },
    "rtx6000|small": {
      slug: "deepseek-ai/DeepSeek-V4-Flash",
      tp: 4,
      multinode: false
    }
  };
  const PD_TP_SPEC = {
    "b200|small": {
      tp: 2,
      multinode: false
    },
    "b200|big": {
      tp: 8,
      multinode: false
    },
    "gb300|small": {
      tp: 4,
      multinode: false
    },
    "gb300|big": {
      tp: 4,
      multinode: false
    },
    "gb200|small": {
      tp: 4,
      multinode: false
    },
    "gb200|big": {
      tp: 8,
      multinode: true,
      nnodes: 2
    },
    "h200|small": {
      tp: 4,
      multinode: false
    },
    "h200|big": {
      tp: 16,
      multinode: true,
      nnodes: 2
    }
  };
  const VERIFIED_RECIPES = new Set(["b200|small|low-latency", "b200|small|balanced", "b200|small|max-throughput", "b200|small|cp", "b200|small|pd-disagg", "b200|big|low-latency", "b200|big|balanced", "b200|big|max-throughput", "b200|big|cp", "h200|small|low-latency", "h200|small|balanced", "h200|small|max-throughput", "gb300|small|low-latency", "gb300|big|low-latency", "gb300|small|balanced", "gb300|big|balanced", "gb300|small|max-throughput", "gb300|big|max-throughput", "h200|small|cp", "h200|small|pd-disagg", "h200|big|low-latency", "h200|big|balanced", "h200|big|max-throughput", "h200|big|pd-disagg", "gb300|small|cp", "gb300|big|cp", "gb300|small|pd-disagg", "gb300|big|pd-disagg", "gb200|small|low-latency", "gb200|small|balanced", "gb200|small|max-throughput", "gb200|small|cp", "gb200|big|low-latency", "gb200|big|balanced", "gb200|big|max-throughput", "h200-fp4|small|low-latency", "h200-fp4|small|balanced", "h200-fp4|small|max-throughput", "h200-fp4|big|low-latency", "h200-fp4|big|balanced", "h200-fp4|big|max-throughput", "h100|small|low-latency", "h100|small|balanced", "h100|small|max-throughput", "h100|big|low-latency", "h100|big|balanced", "h100|big|max-throughput", "h100-fp8|small|low-latency", "h100-fp8|small|balanced", "h100-fp8|small|max-throughput", "rtx6000|small|low-latency"]);
  const TBD_RECIPES = new Set(["h200|big|cp", "gb200|small|pd-disagg", "gb200|big|pd-disagg"]);
  const TBD_PLACEHOLDER = "# to be provided";
  const BEING_VERIFIED_NOTE = "# NOTE: this recipe is being verified on the latest checkpoint";
  const commentOutCommand = cmd => cmd.split("\n").map(line => line.length ? `# ${line}` : "#").join("\n");
  const DEEPEP_LARGE_SMS_FLAG = `  --deepep-config '{"normal_dispatch":{"num_sms":96},"normal_combine":{"num_sms":96}}'`;
  const multiNodeFlags = nnodes => [`  --nnodes ${nnodes}`, `  --node-rank <node-rank>`, `  --dist-init-addr <node0-ip>:20000`];
  const prependMultiNodeNote = (cmd, nnodes) => `# Multi-node (${nnodes} nodes). Run the same command on every node with:\n` + `#   <node-rank> = 0 on the head node, 1..${nnodes - 1} on the others\n` + `#   <node0-ip>  = IP of the head node (reachable from all others)\n` + `${cmd}`;
  const isHopperFp8 = effHwId => effHwId === "h200" || effHwId === "h100-fp8";
  const generateCommand = () => {
    const {hardware: userHardware, modelSize, quantization, recipe, reasoningParser, toolcall, hicache, megamoe} = values;
    const rawHardware = userHardware === "b300" ? "b200" : userHardware;
    const hardware = effHw(rawHardware, quantization);
    const specKey = `${hardware}|${modelSize}`;
    const spec = HW_SIZE_SPEC[specKey];
    const {slug, tp, multinode, nnodes} = spec;
    const isBig = modelSize === "big";
    if (recipe === "pd-disagg") {
      return buildPDDisaggCommand(hardware, modelSize);
    }
    if (hardware === "rtx6000") {
      const verifyKey = `${hardware}|${modelSize}|${recipe}`;
      const rtx6000Flags = ["  --trust-remote-code", `  --model-path ${slug}`, `  --tp ${tp}`, "  --moe-runner-backend marlin", "  --mem-fraction-static 0.70", "  --cuda-graph-max-bs 32"];
      if (toolcall === "enabled") rtx6000Flags.push("  --tool-call-parser deepseekv4");
      if (reasoningParser === "enabled") rtx6000Flags.push("  --reasoning-parser deepseek-v4");
      rtx6000Flags.push("  --host 0.0.0.0");
      rtx6000Flags.push("  --port 30000");
      const rtx6000Cmd = `sglang serve \\\n${rtx6000Flags.join(" \\\n")}`;
      return VERIFIED_RECIPES.has(verifyKey) ? rtx6000Cmd : `${BEING_VERIFIED_NOTE}\n${commentOutCommand(rtx6000Cmd)}`;
    }
    if (hardware === "h200-fp4") {
      const verifyKey = `${hardware}|${modelSize}|${recipe}`;
      if (TBD_RECIPES.has(verifyKey)) return TBD_PLACEHOLDER;
      const useFlashinferMxfp4 = isBig || recipe === "balanced";
      const fp4Flags = ["  --trust-remote-code", `  --model-path ${slug}`, `  --tp ${tp}`, useFlashinferMxfp4 ? "  --moe-runner-backend flashinfer_mxfp4" : "  --moe-runner-backend marlin"];
      if (recipe === "low-latency") {
        fp4Flags.push("  --speculative-algo EAGLE");
        fp4Flags.push("  --speculative-num-steps 3");
        fp4Flags.push("  --speculative-eagle-topk 1");
        fp4Flags.push("  --speculative-num-draft-tokens 4");
      } else if (recipe === "balanced") {
        fp4Flags.push("  --speculative-algo EAGLE");
        fp4Flags.push("  --speculative-num-steps 1");
        fp4Flags.push("  --speculative-eagle-topk 1");
        fp4Flags.push("  --speculative-num-draft-tokens 2");
      }
      if (isBig) {
        fp4Flags.push(recipe === "low-latency" ? "  --mem-fraction-static 0.83" : "  --mem-fraction-static 0.88");
      }
      if (toolcall === "enabled") fp4Flags.push("  --tool-call-parser deepseekv4");
      if (reasoningParser === "enabled") fp4Flags.push("  --reasoning-parser deepseek-v4");
      if (hicache === "l2") {
        fp4Flags.push("  --enable-hierarchical-cache");
        fp4Flags.push("  --hicache-ratio 2");
        fp4Flags.push("  --hicache-size 0");
        fp4Flags.push("  --hicache-write-policy write_through");
        fp4Flags.push("  --hicache-io-backend direct");
        fp4Flags.push("  --hicache-mem-layout page_first_direct");
      }
      fp4Flags.push("  --host 0.0.0.0");
      fp4Flags.push("  --port 30000");
      const fp4Env = [];
      if (hicache === "l2") fp4Env.push("SGLANG_ENABLE_UNIFIED_RADIX_TREE=1");
      const fp4EnvBlock = fp4Env.length ? fp4Env.join(" \\\n") + " \\\n" : "";
      const fp4Cmd = `${fp4EnvBlock}sglang serve \\\n${fp4Flags.join(" \\\n")}`;
      return VERIFIED_RECIPES.has(verifyKey) ? fp4Cmd : `${BEING_VERIFIED_NOTE}\n${commentOutCommand(fp4Cmd)}`;
    }
    if (hardware === "h100") {
      const verifyKey = `${hardware}|${modelSize}|${recipe}`;
      if (TBD_RECIPES.has(verifyKey)) return TBD_PLACEHOLDER;
      const h100Env = isBig ? ["SGLANG_SHARED_EXPERT_TP1=1"] : [];
      const h100EnvBlock = h100Env.length ? h100Env.join(" \\\n") + " \\\n" : "";
      const h100Flags = ["  --trust-remote-code", `  --model-path ${slug}`, `  --tp ${tp}`];
      if (multinode) h100Flags.push(...multiNodeFlags(nnodes));
      h100Flags.push("  --moe-runner-backend marlin");
      if (recipe === "low-latency") {
        h100Flags.push("  --speculative-algo EAGLE");
        h100Flags.push("  --speculative-num-steps 3");
        h100Flags.push("  --speculative-eagle-topk 1");
        h100Flags.push("  --speculative-num-draft-tokens 4");
      } else if (recipe === "balanced") {
        h100Flags.push("  --speculative-algo EAGLE");
        h100Flags.push("  --speculative-num-steps 1");
        h100Flags.push("  --speculative-eagle-topk 1");
        h100Flags.push("  --speculative-num-draft-tokens 2");
      }
      if (isBig) {
        h100Flags.push("  --mem-fraction-static 0.9");
        if (recipe !== "max-throughput") {
          h100Flags.push("  --cuda-graph-max-bs 8");
          h100Flags.push("  --max-running-requests 32");
        }
      }
      if (toolcall === "enabled") h100Flags.push("  --tool-call-parser deepseekv4");
      if (reasoningParser === "enabled") h100Flags.push("  --reasoning-parser deepseek-v4");
      h100Flags.push("  --host 0.0.0.0");
      h100Flags.push("  --port 30000");
      const h100Cmd = `${h100EnvBlock}sglang serve \\\n${h100Flags.join(" \\\n")}`;
      const h100WithNote = multinode ? prependMultiNodeNote(h100Cmd, nnodes) : h100Cmd;
      return VERIFIED_RECIPES.has(verifyKey) ? h100WithNote : `${BEING_VERIFIED_NOTE}\n${commentOutCommand(h100WithNote)}`;
    }
    const HW_ENV = ({
      h200: ["SGLANG_DSV4_FP4_EXPERTS=0"],
      "h100-fp8": ["SGLANG_DSV4_FP4_EXPERTS=0"],
      b200: [],
      gb300: [],
      gb200: multinode ? ["NCCL_MNNVL_ENABLE=1", "NCCL_CUMEM_ENABLE=1"] : []
    })[hardware];
    const recipeEnv = [];
    if (recipe === "low-latency") {
      if (hardware === "h200" && isBig) {
        recipeEnv.push("SGLANG_DEEPEP_NUM_MAX_DISPATCH_TOKENS_PER_RANK=128");
      } else if (hardware === "gb200" && isBig) {
        recipeEnv.push("SGLANG_DEEPEP_NUM_MAX_DISPATCH_TOKENS_PER_RANK=256");
      }
    } else if (recipe === "balanced") {
      if (isHopperFp8(hardware)) {
        recipeEnv.push(isBig ? "SGLANG_DEEPEP_NUM_MAX_DISPATCH_TOKENS_PER_RANK=128" : "SGLANG_DEEPEP_NUM_MAX_DISPATCH_TOKENS_PER_RANK=256");
      } else if (isBig && hardware === "b200") {} else {
        recipeEnv.push(isBig ? "SGLANG_DEEPEP_NUM_MAX_DISPATCH_TOKENS_PER_RANK=256" : "SGLANG_DEEPEP_NUM_MAX_DISPATCH_TOKENS_PER_RANK=1024");
      }
    } else if (recipe === "max-throughput") {
      if (isHopperFp8(hardware)) {
        recipeEnv.push(isBig ? "SGLANG_DEEPEP_NUM_MAX_DISPATCH_TOKENS_PER_RANK=128" : "SGLANG_DEEPEP_NUM_MAX_DISPATCH_TOKENS_PER_RANK=256");
      } else {
        recipeEnv.push(isBig ? "SGLANG_DEEPEP_NUM_MAX_DISPATCH_TOKENS_PER_RANK=256" : "SGLANG_DEEPEP_NUM_MAX_DISPATCH_TOKENS_PER_RANK=1024");
      }
    } else if (recipe === "cp") {
      if (hardware === "h200") {
        recipeEnv.push("SGLANG_DEEPEP_NUM_MAX_DISPATCH_TOKENS_PER_RANK=1024");
      } else {
        recipeEnv.push(isBig ? "SGLANG_DEEPEP_NUM_MAX_DISPATCH_TOKENS_PER_RANK=256" : "SGLANG_DEEPEP_NUM_MAX_DISPATCH_TOKENS_PER_RANK=1024");
      }
    }
    const flags = [];
    flags.push("  --trust-remote-code");
    flags.push(`  --model-path ${slug}`);
    if (recipe === "low-latency") {
      flags.push(`  --tp ${tp}`);
      if (hardware === "h200" && isBig) {
        flags.push(`  --dp ${tp}`);
        flags.push("  --enable-dp-attention");
      }
      if (multinode) flags.push(...multiNodeFlags(nnodes));
      if (hardware === "h200" && isBig) {
        flags.push("  --moe-a2a-backend deepep");
      }
      if (!isHopperFp8(hardware)) {
        flags.push("  --moe-runner-backend flashinfer_mxfp4");
      }
      if (hardware === "h200" && isBig) {
        flags.push("  --cuda-graph-max-bs 8");
        flags.push("  --max-running-requests 32");
      }
      flags.push("  --speculative-algo EAGLE");
      flags.push("  --speculative-num-steps 3");
      flags.push("  --speculative-eagle-topk 1");
      flags.push("  --speculative-num-draft-tokens 4");
      if (!isHopperFp8(hardware)) {
        flags.push(isBig ? "  --chunked-prefill-size 8192" : "  --chunked-prefill-size 4096");
        flags.push("  --disable-flashinfer-autotune");
        flags.push("  --swa-full-tokens-ratio 0.1");
      }
      if (isBig && !isHopperFp8(hardware)) {
        flags.push("  --mem-fraction-static 0.90");
      } else if (isBig) {
        flags.push("  --mem-fraction-static 0.88");
      }
    } else if (recipe === "balanced") {
      flags.push(`  --tp ${tp}`);
      flags.push(`  --dp ${tp}`);
      flags.push("  --enable-dp-attention");
      if (multinode) flags.push(...multiNodeFlags(nnodes));
      if (isBig && hardware === "b200") {
        flags.push("  --moe-runner-backend flashinfer_mxfp4");
        flags.push("  --disable-flashinfer-autotune");
        flags.push("  --chunked-prefill-size 32768");
        flags.push("  --swa-full-tokens-ratio 0.1");
      } else {
        flags.push("  --moe-a2a-backend deepep");
      }
      flags.push("  --speculative-algo EAGLE");
      flags.push("  --speculative-num-steps 1");
      flags.push("  --speculative-eagle-topk 1");
      flags.push("  --speculative-num-draft-tokens 2");
      if (hardware === "h200" && isBig) {
        flags.push("  --mem-fraction-static 0.88");
      } else if (isBig && hardware === "gb300") {
        flags.push("  --mem-fraction-static 0.9");
      } else if (isBig && hardware === "gb200") {
        flags.push("  --mem-fraction-static 0.78");
      } else if (isBig) {
        flags.push("  --mem-fraction-static 0.92");
      }
      if (hardware === "h200" && isBig) {
        flags.push("  --cuda-graph-max-bs 8");
        flags.push("  --max-running-requests 32");
      } else if (hardware === "h200") {
        flags.push("  --cuda-graph-max-bs 128");
        flags.push("  --max-running-requests 128");
      } else if (isBig && hardware === "b200") {
        flags.push("  --cuda-graph-max-bs 256");
      } else if (isBig && hardware === "gb300") {
        flags.push("  --cuda-graph-max-bs 128");
        flags.push("  --max-running-requests 256");
      } else if (isBig && hardware === "gb200") {
        flags.push("  --cuda-graph-max-bs 64");
        flags.push("  --max-running-requests 128");
      }
      if (!multinode && megamoe === "disabled") flags.push(DEEPEP_LARGE_SMS_FLAG);
    } else if (recipe === "max-throughput") {
      flags.push(`  --tp ${tp}`);
      flags.push(`  --dp ${tp}`);
      flags.push("  --enable-dp-attention");
      if (multinode) flags.push(...multiNodeFlags(nnodes));
      flags.push("  --moe-a2a-backend deepep");
      if (hardware === "h200" && isBig) {
        flags.push("  --mem-fraction-static 0.88");
      } else if (isBig && hardware === "gb300") {
        flags.push("  --mem-fraction-static 0.9");
      } else if (isBig && hardware === "gb200") {
        flags.push("  --mem-fraction-static 0.78");
      } else if (isBig) {
        flags.push("  --mem-fraction-static 0.835");
      }
      if (hardware === "h200") {
        flags.push("  --cuda-graph-max-bs 128");
        flags.push("  --max-running-requests 256");
      } else if (isBig && hardware === "b200") {
        flags.push("  --cuda-graph-max-bs 544");
        flags.push("  --swa-full-tokens-ratio 0.075");
        flags.push("  --chunked-prefill-size 65536");
        flags.push("  --tokenizer-worker-num 8");
        flags.push("  --enable-prefill-delayer");
      } else if (isBig && hardware === "gb300") {
        flags.push("  --cuda-graph-max-bs 128");
        flags.push("  --max-running-requests 256");
      } else if (isBig && hardware === "gb200") {
        flags.push("  --cuda-graph-max-bs 64");
        flags.push("  --max-running-requests 256");
      }
      if (!multinode && megamoe === "disabled") flags.push(DEEPEP_LARGE_SMS_FLAG);
    } else if (recipe === "cp") {
      flags.push(`  --tp ${tp}`);
      if (multinode) flags.push(...multiNodeFlags(nnodes));
      flags.push("  --moe-a2a-backend deepep");
      flags.push("  --enable-nsa-prefill-context-parallel");
      flags.push("  --nsa-prefill-cp-mode round-robin-split");
      flags.push("  --chunked-prefill-size 16384");
      if (hardware === "gb300" && isBig) {
        flags.push("  --mem-fraction-static 0.88");
      } else {
        flags.push("  --mem-fraction-static 0.78");
      }
      if (isBig && hardware !== "h200") {
        flags.push("  --cuda-graph-max-bs 256");
        flags.push("  --max-running-requests 256");
      } else {
        flags.push("  --max-running-requests 1024");
      }
      if (!multinode) flags.push(DEEPEP_LARGE_SMS_FLAG);
    }
    if (toolcall === "enabled") flags.push("  --tool-call-parser deepseekv4");
    if (reasoningParser === "enabled") flags.push("  --reasoning-parser deepseek-v4");
    if (hicache === "l2") {
      flags.push("  --enable-hierarchical-cache");
      flags.push("  --hicache-ratio 2");
      flags.push("  --hicache-size 0");
      flags.push("  --hicache-write-policy write_through");
      flags.push("  --hicache-io-backend direct");
      flags.push("  --hicache-mem-layout page_first_direct");
    }
    if (megamoe !== "disabled") {
      const idx = flags.indexOf("  --moe-a2a-backend deepep");
      if (idx !== -1) {
        flags[idx] = "  --moe-a2a-backend megamoe";
      } else {
        flags.push("  --moe-a2a-backend megamoe");
      }
    }
    flags.push("  --host 0.0.0.0");
    flags.push("  --port 30000");
    const hicacheEnv = [];
    if (hicache === "l2") {
      hicacheEnv.push("SGLANG_ENABLE_UNIFIED_RADIX_TREE=1");
    }
    const megamoeEnv = [];
    if (megamoe !== "disabled" && recipe === "max-throughput") {
      megamoeEnv.push("SGLANG_OPT_DEEPGEMM_MEGA_MOE_NUM_MAX_TOKENS_PER_RANK=8320");
    }
    if (megamoe === "w4a4") {
      megamoeEnv.push("SGLANG_OPT_DEEPGEMM_MEGA_MOE_USE_FP4_ACTS=1");
      megamoeEnv.push("SGLANG_OPT_DEEPGEMM_MEGA_MOE_USE_MXF4_KIND=1");
    }
    const filteredRecipeEnv = megamoe !== "disabled" ? recipeEnv.filter(e => !e.startsWith("SGLANG_DEEPEP_NUM_MAX_DISPATCH_TOKENS_PER_RANK=")) : recipeEnv;
    const envAll = [...HW_ENV, ...filteredRecipeEnv, ...hicacheEnv, ...megamoeEnv];
    const envBlock = envAll.length ? envAll.join(" \\\n") + " \\\n" : "";
    const simplifyNote = isBig && hardware === "b200" && recipeEnv.length > 2 ? "# flags will be simplified\n" : "";
    const base = `${simplifyNote}${envBlock}sglang serve \\\n${flags.join(" \\\n")}`;
    let cmd = base;
    if (recipe === "cp") {
      cmd = `# NOTE: --enable-nsa-prefill-context-parallel / --nsa-prefill-cp-mode were\n` + `# renamed to --enable-dsa-prefill-context-parallel / --dsa-prefill-cp-mode\n` + `# in PR #25821 (merged 2026-05-20). The cookbook emits the old nsa-* names\n` + `# because the :latest release image predates that PR. If you're running\n` + `# SGLang built from main, replace nsa- with dsa- in the two flags below.\n` + cmd;
    }
    if (hardware === "gb200" && multinode) {
      cmd = `# The following env vars may be needed depending on your cluster:\n` + `#   GLOO_SOCKET_IFNAME=<your-nic>\n` + `#   NVSHMEM_ENABLE_NIC_PE_MAPPING=1\n` + `#   NVSHMEM_HCA_LIST=<your-hca-list>\n` + cmd;
    }
    if (hardware === "gb200" && isBig && megamoe === "disabled" && flags.some(f => f.includes("--moe-a2a-backend deepep"))) {
      cmd = `# NOTE: for the DeepEP backend, use the cu129 docker image\n` + `# (lmsysorg/sglang:latest-cu129) instead of the default \`:latest\`.\n` + cmd;
    }
    const withMultinode = multinode ? prependMultiNodeNote(cmd, nnodes) : cmd;
    if (hardware === "h200" && isBig && recipe === "low-latency") {
      const singleFlags = ["  --trust-remote-code", "  --model-path deepseek-ai/DeepSeek-V4-Pro", "  --tp 8", "  --moe-runner-backend marlin", "  --speculative-algo EAGLE", "  --speculative-num-steps 3", "  --speculative-eagle-topk 1", "  --speculative-num-draft-tokens 4", "  --chunked-prefill-size 4096", "  --disable-flashinfer-autotune", "  --mem-fraction-static 0.88"];
      if (toolcall === "enabled") singleFlags.push("  --tool-call-parser deepseekv4");
      if (reasoningParser === "enabled") singleFlags.push("  --reasoning-parser deepseek-v4");
      singleFlags.push("  --host 0.0.0.0");
      singleFlags.push("  --port 30000");
      const singleNodeCmd = `sglang serve \\\n${singleFlags.join(" \\\n")}`;
      const combined = `# --- Single-Node (TP=8, Marlin) ---\n${singleNodeCmd}\n\n` + `# --- Multi-Node (2 nodes, TP=16, DP-Attn + DeepEP) ---\n${withMultinode}`;
      const verifyKey = `${hardware}|${modelSize}|${recipe}`;
      if (TBD_RECIPES.has(verifyKey)) return TBD_PLACEHOLDER;
      return VERIFIED_RECIPES.has(verifyKey) ? combined : `${BEING_VERIFIED_NOTE}\n${commentOutCommand(combined)}`;
    }
    const verifyKey = `${hardware}|${modelSize}|${recipe}`;
    if (TBD_RECIPES.has(verifyKey)) return TBD_PLACEHOLDER;
    return VERIFIED_RECIPES.has(verifyKey) ? withMultinode : `${BEING_VERIFIED_NOTE}\n${commentOutCommand(withMultinode)}`;
  };
  const buildPDDisaggCommand = (rawHardware, modelSize) => {
    const hardware = rawHardware === "b300" ? "b200" : rawHardware;
    const specKey = `${hardware}|${modelSize}`;
    const {tp: pdTp, multinode, nnodes} = PD_TP_SPEC[specKey];
    const slug = HW_SIZE_SPEC[specKey].slug;
    const ibDevice = ({
      h200: "mlx5_0",
      b200: "mlx5_7",
      gb300: "",
      gb200: ""
    })[hardware];
    const isGB300 = hardware === "gb300";
    const isBlackwell = hardware === "b200" || hardware === "gb200" || isGB300;
    const HW_ENV = ({
      h200: ["SGLANG_DSV4_FP4_EXPERTS=0"],
      b200: [],
      gb300: [],
      gb200: []
    })[hardware];
    const MNNVL_ENV = isGB300 ? ["SGLANG_MOONCAKE_CUSTOM_MEM_POOL=True"] : [];
    const buildRole = (mode, port, distPort) => {
      const roleEnv = [];
      if (hardware === "b200" && mode === "decode") {
        roleEnv.push("SGLANG_DEEPEP_NUM_MAX_DISPATCH_TOKENS_PER_RANK=1024");
      }
      if (isGB300) {
        roleEnv.push(modelSize === "big" ? "SGLANG_DEEPEP_NUM_MAX_DISPATCH_TOKENS_PER_RANK=256" : "SGLANG_DEEPEP_NUM_MAX_DISPATCH_TOKENS_PER_RANK=1024");
      }
      if (hardware === "h200" && modelSize === "big") {
        roleEnv.push("SGLANG_DEEPEP_NUM_MAX_DISPATCH_TOKENS_PER_RANK=128");
      }
      const envAll = [...HW_ENV, ...roleEnv, ...MNNVL_ENV];
      const envBlock = envAll.length ? envAll.join(" \\\n") + " \\\n" : "";
      const flags = [];
      flags.push("  --trust-remote-code");
      flags.push(`  --model-path ${slug}`);
      flags.push(`  --tp ${pdTp}`);
      flags.push(`  --dp ${pdTp}`);
      flags.push("  --enable-dp-attention");
      if (multinode) flags.push(...multiNodeFlags(nnodes));
      if (isBlackwell || hardware === "h200" && modelSize === "big") {
        flags.push("  --moe-a2a-backend deepep");
      }
      flags.push(`  --disaggregation-mode ${mode}`);
      flags.push("  --disaggregation-transfer-backend mooncake");
      if (ibDevice) flags.push(`  --disaggregation-ib-device ${ibDevice}`);
      if (!isGB300 && !multinode) flags.push(`  --dist-init-addr 127.0.0.1:${distPort}`);
      if (hardware === "h200" && modelSize === "big") {
        flags.push("  --cuda-graph-max-bs 128");
        flags.push("  --mem-fraction-static 0.9");
      }
      if (mode === "decode") {
        if (isGB300 && modelSize === "big") {
          flags.push("  --max-running-requests 128");
          flags.push("  --mem-fraction-static 0.9");
          flags.push("  --cuda-graph-max-bs 128");
        } else {
          flags.push("  --max-running-requests 256");
        }
        if (values.toolcall === "enabled") flags.push("  --tool-call-parser deepseekv4");
        if (values.reasoningParser === "enabled") flags.push("  --reasoning-parser deepseek-v4");
      }
      flags.push("  --host 0.0.0.0");
      flags.push(`  --port ${port}`);
      return `${envBlock}sglang serve \\\n${flags.join(" \\\n")}`;
    };
    const prefillHeader = multinode ? `# --- Prefill role (port 30000) — multi-node, run on each of ${nnodes} nodes ---` : "# --- Prefill role (port 30000) ---";
    const decodeHeader = multinode ? `# --- Decode role (port 30001) — multi-node, run on each of ${nnodes} nodes ---` : "# --- Decode role (port 30001) ---";
    const prefill = `${prefillHeader}\n${buildRole("prefill", 30000, 30335)}`;
    const decode = `${decodeHeader}\n${buildRole("decode", 30001, 30435)}`;
    const router = `# --- Router (port 8000) ---
python3 -m sglang_router.launch_router \\
  --pd-disaggregation \\
  --prefill http://<prefill-host>:30000 \\
  --decode http://<decode-host>:30001 \\
  --host 0.0.0.0 --port 8000 \\
  --disable-circuit-breaker \\
  --health-check-interval-secs 999999`;
    const full = `${prefill}\n\n${decode}\n\n${router}`;
    const verifyKey = `${hardware}|${modelSize}|pd-disagg`;
    if (TBD_RECIPES.has(verifyKey)) return TBD_PLACEHOLDER;
    return VERIFIED_RECIPES.has(verifyKey) ? full : `${BEING_VERIFIED_NOTE}\n${commentOutCommand(full)}`;
  };
  const containerStyle = {
    maxWidth: "900px",
    margin: "0 auto",
    display: "flex",
    flexDirection: "column",
    gap: "4px"
  };
  const cardStyle = {
    padding: "8px 12px",
    border: `1px solid ${isDark ? "#374151" : "#e5e7eb"}`,
    borderLeft: `3px solid ${isDark ? "#E85D4D" : "#D45D44"}`,
    borderRadius: "4px",
    display: "flex",
    alignItems: "center",
    gap: "12px",
    background: isDark ? "#1f2937" : "#fff"
  };
  const titleStyle = {
    fontSize: "13px",
    fontWeight: "600",
    minWidth: "140px",
    flexShrink: 0,
    color: isDark ? "#e5e7eb" : "inherit"
  };
  const itemsStyle = {
    display: "flex",
    rowGap: "2px",
    columnGap: "6px",
    flexWrap: "wrap",
    alignItems: "center",
    flex: 1
  };
  const labelBaseStyle = {
    padding: "4px 10px",
    border: `1px solid ${isDark ? "#9ca3af" : "#d1d5db"}`,
    borderRadius: "3px",
    cursor: "pointer",
    display: "inline-flex",
    flexDirection: "column",
    alignItems: "center",
    justifyContent: "center",
    fontWeight: "500",
    fontSize: "13px",
    transition: "all 0.2s",
    userSelect: "none",
    minWidth: "45px",
    textAlign: "center",
    flex: 1,
    background: isDark ? "#374151" : "#fff",
    color: isDark ? "#e5e7eb" : "inherit"
  };
  const checkedStyle = {
    background: "#D45D44",
    color: "white",
    borderColor: "#D45D44"
  };
  const disabledStyle = {
    cursor: "not-allowed",
    opacity: 0.4
  };
  const subtitleStyle = {
    display: "block",
    fontSize: "9px",
    marginTop: "1px",
    lineHeight: "1.1",
    opacity: 0.7
  };
  const commandDisplayStyle = {
    flex: 1,
    padding: "12px 16px",
    background: isDark ? "#111827" : "#f5f5f5",
    borderRadius: "6px",
    fontFamily: "'Menlo', 'Monaco', 'Courier New', monospace",
    fontSize: "12px",
    lineHeight: "1.5",
    color: isDark ? "#e5e7eb" : "#374151",
    whiteSpace: "pre-wrap",
    overflowX: "auto",
    margin: 0,
    border: `1px solid ${isDark ? "#374151" : "#e5e7eb"}`
  };
  return <div style={containerStyle} className="not-prose">
      {Object.entries(options).map(([key, option]) => {
    const items = resolveItems(option, values);
    return <div key={key} style={cardStyle}>
            <div style={titleStyle}>{option.title}</div>
            <div style={itemsStyle}>
              {items.map(item => {
      const isChecked = values[option.name] === item.id;
      const isDisabled = !!item.disabled;
      return <label key={item.id} style={{
        ...labelBaseStyle,
        ...isChecked ? checkedStyle : {},
        ...isDisabled ? disabledStyle : {}
      }} title={item.disabledReason || ""}>
                    <input type="radio" name={option.name} value={item.id} checked={isChecked} disabled={isDisabled} onChange={() => !isDisabled && handleRadioChange(option.name, item.id)} style={{
        display: "none"
      }} />
                    {item.label}
                    {item.subtitle && <small style={{
        ...subtitleStyle,
        color: isChecked ? "rgba(255,255,255,0.85)" : "inherit"
      }}>
                        {item.subtitle}
                      </small>}
                  </label>;
    })}
            </div>
          </div>;
  })}
      <div style={cardStyle}>
        <div style={titleStyle}>Run this Command:</div>
        <pre style={commandDisplayStyle}>{generateCommand()}</pre>
      </div>
    </div>;
};

## 1. Model Introduction

**DeepSeek-V4** is the next-generation Mixture-of-Experts model from DeepSeek, released 2026-04-24 under an **MIT License**. It ships as two Instruct repos (one per variant) plus matching Base repos:

<table style={{width: "100%", borderCollapse: "collapse", tableLayout: "fixed"}}>
  <colgroup>
    <col style={{width: "30%"}} />

    <col style={{width: "15%"}} />

    <col style={{width: "15%"}} />

    <col style={{width: "40%"}} />
  </colgroup>

  <thead>
    <tr style={{borderBottom: "2px solid #d55816"}}>
      <th style={{textAlign: "left", padding: "10px 12px", fontWeight: 700, whiteSpace: "nowrap", backgroundColor: "rgba(255,255,255,0.02)"}}>Variant</th>
      <th style={{textAlign: "right", padding: "10px 12px", fontWeight: 700, whiteSpace: "nowrap", backgroundColor: "rgba(255,255,255,0.05)"}}>Total params</th>
      <th style={{textAlign: "right", padding: "10px 12px", fontWeight: 700, whiteSpace: "nowrap", backgroundColor: "rgba(255,255,255,0.02)"}}>Active (MoE)</th>
      <th style={{textAlign: "left", padding: "10px 12px", fontWeight: 700, whiteSpace: "nowrap", backgroundColor: "rgba(255,255,255,0.05)"}}>Use</th>
    </tr>
  </thead>

  <tbody>
    <tr>
      <td style={{padding: "9px 12px", fontWeight: 500, backgroundColor: "rgba(255,255,255,0.02)"}}><strong><a href="https://huggingface.co/deepseek-ai/DeepSeek-V4-Flash">DeepSeek-V4-Flash</a></strong></td>
      <td style={{padding: "9px 12px", textAlign: "right", backgroundColor: "rgba(255,255,255,0.05)"}}><strong>284B</strong></td>
      <td style={{padding: "9px 12px", textAlign: "right", backgroundColor: "rgba(255,255,255,0.02)"}}>13B</td>
      <td style={{padding: "9px 12px", backgroundColor: "rgba(255,255,255,0.05)"}}>single-node serving: B200 / GB200 / GB300 / H200 on 4 GPUs</td>
    </tr>

    <tr>
      <td style={{padding: "9px 12px", fontWeight: 500, backgroundColor: "rgba(255,255,255,0.02)"}}><strong><a href="https://huggingface.co/deepseek-ai/DeepSeek-V4-Pro">DeepSeek-V4-Pro</a></strong></td>
      <td style={{padding: "9px 12px", textAlign: "right", backgroundColor: "rgba(255,255,255,0.05)"}}><strong>1.6T</strong></td>
      <td style={{padding: "9px 12px", textAlign: "right", backgroundColor: "rgba(255,255,255,0.02)"}}>49B</td>
      <td style={{padding: "9px 12px", backgroundColor: "rgba(255,255,255,0.05)"}}>high-capacity: B200 8 GPU / GB200 8 GPU (2 nodes) / GB300 4 GPU / H200 8 GPU (FP4) or 16 GPU (SGLang FP8)</td>
    </tr>
  </tbody>
</table>

The Instruct repos ship **FP4 MoE experts + FP8 attention / dense** (one mixed-precision checkpoint covers all GPUs that support FP4). The Base (pre-trained only) variants — `DeepSeek-V4-Flash-Base`, `DeepSeek-V4-Pro-Base` — ship pure FP8 mixed and are **not** for chat / tool calling.

**Key Features** (per the official model card):

* **Hybrid Attention Architecture** — combines Compressed Sparse Attention (CSA) and Heavily Compressed Attention (HCA) for long-context efficiency. At 1M-token context, DeepSeek-V4-Pro uses only \~27% of per-token inference FLOPs and \~10% of KV cache compared with DeepSeek-V3.2.
* **Manifold-Constrained Hyper-Connections (mHC)** — strengthens residual connections, improving signal-propagation stability across layers while preserving expressivity.
* **Muon optimizer** — faster convergence and greater training stability.
* **Context length: 1M tokens**; pre-trained on 32T+ diverse, high-quality tokens.
* **Three reasoning modes**: *Non-think* (fast, intuitive responses), *Think High* (conscious logical analysis, slower but more accurate), *Think Max* (push reasoning to its fullest extent). Recommend a ≥ 384K context window when running Think Max.
* Ships with a dedicated `encoding_dsv4.encode_messages` Python encoder + DSML tool-call grammar (`<｜DSML｜tool_calls>` / `<｜DSML｜invoke>` / `<｜DSML｜parameter>`).

**Recommended Generation Parameters:** `temperature=1.0`, `top_p=1.0` (per the official model card).

**License:** MIT.

**Resources:**

* HuggingFace: [DeepSeek-V4-Flash](https://huggingface.co/deepseek-ai/DeepSeek-V4-Flash), [DeepSeek-V4-Pro](https://huggingface.co/deepseek-ai/DeepSeek-V4-Pro)
* ModelScope: [DeepSeek-V4-Flash](https://modelscope.cn/models/deepseek-ai/DeepSeek-V4-Flash), [DeepSeek-V4-Pro](https://modelscope.cn/models/deepseek-ai/DeepSeek-V4-Pro)

## 2. SGLang Installation

SGLang offers multiple installation methods. Choose based on your hardware platform.

Please refer to the [official SGLang installation guide](../../../docs/get-started/install) for installation instructions.

**Docker Image:** Use `lmsysorg/sglang:latest` for all supported hardware platforms (B300 / B200 / GB200 / GB300 / H200 / H100).

```bash Command theme={null}
docker pull lmsysorg/sglang:latest
```

For how to actually launch the image, see [Install → Method 3: Using Docker](../../../docs/get-started/install#method-3-using-docker). A minimal example (substitute the inner `sglang serve ...` with whatever the [command generator](#3-model-deployment) below produces):

```bash Command theme={null}
docker run --gpus all \
    --shm-size 32g \
    -p 30000:30000 \
    -v ~/.cache/huggingface:/root/.cache/huggingface \
    --env "HF_TOKEN=<your-hf-token>" \
    --ipc=host \
    lmsysorg/sglang:latest \
    sglang serve <use args below>
```

## 3. Model Deployment

SGLang supports three main serving recipes for DeepSeek-V4 with different latency/throughput trade-offs (`low-latency`, `balanced`, `max-throughput`), plus specialized recipes for long-context (`cp`, prefill context-parallel) and prefill/decode disaggregation (`pd-disagg`). The interactive generator below emits the exact launch command for any `(hardware, variant, recipe)` combination.

### 3.1 Basic Configuration

**Interactive Command Generator**: Use the selector below to generate the deployment command for your hardware + recipe combination.

<DeepSeekV4Deployment />

### 3.2 Configuration Tips

**Concurrency & DeepEP dispatch buffer**

Must hold: `max-running-requests × MTP_draft_tokens ≤ SGLANG_DEEPEP_NUM_MAX_DISPATCH_TOKENS_PER_RANK`. Violating it blows DeepEP's dispatch buffer at steady-state load (`deep_ep.cpp:1105`). When tuning, move `--cuda-graph-max-bs`, `--max-running-requests`, and the env together.

The generator currently picks values on the **conservative** side (mirroring an internal stress-test matrix). They run safely out of the box but likely leave throughput on the table — please tune them up toward your actual workload's peak concurrency and report findings back so the defaults can be revised.

**MTP (Multi-Token Prediction, EAGLE)**

* `low-latency`: steps=3, draft-tokens=4 → largest win at bs=1.
* `balanced`: steps=1, draft-tokens=2 → gentler MTP, reduces throughput hit at higher batch.
* `max-throughput`: MTP disabled — at saturation the verify step costs more than it saves.
* MTP currently requires `SGLANG_ENABLE_SPEC_V2=1`.

**EPLB + DeepEP Waterfill (Experimental)**

For recorded/static EPLB reproduction, first record an expert-distribution file by following
[Capture expert selection distribution in MoE models](../../../docs/basic_usage/native_api.mdx#capture-expert-selection-distribution-in-moe-models).
For reproduction runs, use the generated `expert_distribution_recorder_*.pt` as
the initial expert location. **Please checkout to latest main branch for this feature.**

For non-PD reproduction, use:

```bash Command theme={null}
--moe-a2a-backend deepep \
--deepep-mode auto \
--init-expert-location /path/to/expert_distribution_recorder_*.pt \
--enable-deepep-waterfill
```

For PD-Disagg reproduction, use `normal` mode on the prefill server and
`low_latency` mode on the decode server. Add the same `--init-expert-location`
flag to both commands:

```bash Command theme={null}
# prefill
--moe-a2a-backend deepep \
--deepep-mode normal \
--init-expert-location /path/to/expert_distribution_recorder_*.pt \
--enable-deepep-waterfill

# decode
--moe-a2a-backend deepep \
--deepep-mode low_latency \
--init-expert-location /path/to/expert_distribution_recorder_*.pt \
--enable-deepep-waterfill
```

You can also add `--ep-num-redundant-experts` and `--eplb-algorithm` to customize
EPLB placement.

MegaMoE is not supported with this DeepEP Waterfill recipe yet. Waterfill routes
the shared expert through DeepEP for load balancing, so `--enable-deepep-waterfill`
requires `--moe-a2a-backend deepep`.

<a id="hopper-note" />

**Hopper (H200) note**

We provide two different options for running DeepSeek-V4 models on Hopper devices (H200)

* Original FP4 checkpoints: To run original FP4 checkpoints, we provide two different options for w4a16 MoE kernels: Marlin (`--moe-runner-backend marlin`) and Flashinfer (`--moe-runner-backend flashinfer_mxfp4`). For this variant we only support Tensor Parallelism. Complete Pro model can be run on a single H200 node with this option.
* Converted FP8 checkpoints: We also provide pre-converted FP8 checkpoints (`sgl-project/DeepSeek-V4-Flash-FP8`, `sgl-project/DeepSeek-V4-Pro-FP8`), which support more parallelism and features.

PD-Disagg recipes on H200 may require `docker run --privileged --ulimit memlock=-1`
(or `--device /dev/infiniband:/dev/infiniband --cap-add IPC_LOCK`) so mooncake
can discover the IB HCAs; without IB exposure mooncake silently falls back to
TCP, which can lead to garbled KV transfer on large checkpoints.

**MegaMoE**

MegaMoE fuses expert dispatch + GEMM into a single kernel for higher throughput
on MoE layers. To enable it, use the **MegaMoE** toggle in the
[command generator above](#3-model-deployment) — the generator will swap
`--moe-a2a-backend deepep` for `--moe-a2a-backend megamoe` and add the
relevant env vars automatically.

Two variants are exposed:

* **W4A8** — default MegaMoE kernel (FP4 weights, FP8 activations).
* **W4A4** — adds `SGLANG_OPT_DEEPGEMM_MEGA_MOE_USE_FP4_ACTS=1` and
  `SGLANG_OPT_DEEPGEMM_MEGA_MOE_USE_MXF4_KIND=1` to run the custom W4A4
  kernel (FP4 activations). Higher throughput with negligible accuracy drop
  (\~89.5 GPQA on Pro).

Notes:

* MegaMoE is **not** supported on Hopper (H100 / H200) nor on the `low-latency` / `balanced` / `cp` settings — it is only wired into the `max-throughput` recipe on Blackwell. When running MegaMoE, don't set `--moe-runner-backend` manually.
* Adjust `SGLANG_OPT_DEEPGEMM_MEGA_MOE_NUM_MAX_TOKENS_PER_RANK` based on your workload and memory usage. Setting higher number of tokens for MegaMoE requires more HBM space. (recommended: 8320 for max-throughput).

**GB300 PD-Disagg cross-pod MNNVL**

On some GB300 clusters with cross-pod KV transfer over NVLink, mooncake may
fail with `nvlink_transport.cpp:497 Requested address ... not found!`. If
this happens, prepend `MC_FORCE_MNNVL=1 NCCL_MNNVL_ENABLE=1 NCCL_CUMEM_ENABLE=1`
to both prefill and decode `sglang serve` commands.

## 4. Model Invocation

### 4.1 Basic Usage

For basic API usage and request examples, see:

* [Basic API Usage](../../../docs/basic_usage/send_request)

Once the server is running (for example via the command generator above), send a request:

```shell Command theme={null}
curl http://localhost:30000/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "deepseek-ai/DeepSeek-V4-Flash",
    "messages": [{"role": "user", "content": "What is 15% of 240?"}]
  }'
```

> **PD-Disagg note**: if you deployed with the `pd-disagg` recipe from the generator above, the prefill server is on port `30000`, the decode server on `30001`, and the **router** on port `8000` — client traffic should target `http://localhost:8000`, not `:30000`.

### 4.2 Advanced Usage

#### 4.2.1 Reasoning Parser

Enable the `deepseek-v4` reasoning parser (check the box in the [command panel above](#3-model-deployment)) to separate thinking from the final answer into `reasoning_content` vs `content`.

<Accordion title="Streaming with Thinking Process (Python)">
  ```python Example theme={null}
  from openai import OpenAI

  client = OpenAI(
      base_url="http://localhost:30000/v1",
      api_key="EMPTY"
  )

  response = client.chat.completions.create(
      model="deepseek-ai/DeepSeek-V4-Flash",
      messages=[
          {"role": "user", "content": "Solve this problem step by step: What is 15% of 240?"}
      ],
      max_tokens=2048,
      extra_body={"chat_template_kwargs": {"thinking": True}},
      stream=True,
  )

  thinking_started = False
  has_thinking = False
  has_answer = False

  for chunk in response:
      if not chunk.choices:
          continue
      delta = chunk.choices[0].delta

      if getattr(delta, "reasoning_content", None):
          if not thinking_started:
              print("=============== Thinking =================", flush=True)
              thinking_started = True
          has_thinking = True
          print(delta.reasoning_content, end="", flush=True)

      if delta.content:
          if has_thinking and not has_answer:
              print("\n=============== Content =================", flush=True)
              has_answer = True
          print(delta.content, end="", flush=True)

  print()
  ```
</Accordion>

<Accordion title="Example Output">
  ```text Output theme={null}
  We are asked: "What is 15% of 240?" This is a simple percentage problem. I need to provide a step-by-step solution. The user wants the solution explained step by step. I'll calculate 15% of 240: 0.15 * 240 = 36. I'll break it down into steps: understand what percent means, convert percentage to decimal or fraction, then multiply. I'll present the answer clearly.</think>To find 15% of 240, follow these steps:

  **Step 1: Understand the meaning of percent**
  "Percent" means "per hundred," so 15% means 15 out of every100, or \( \frac{15}{100} \).

  **Step2: Convert the percentage to a decimal or fraction**
  \( 15\% = \frac{15}{100} = 0.15 \)

  **Step3: Multiply by the given number**
  Multiply the decimal form by 240:
  \( 0.15 \times 240 \)

  **Step4: Perform the multiplication**
  \( 0.15 \times 240 = 36 \)

  **Answer:** 15% of 240 is **36**.
  ```
</Accordion>

#### 4.2.2 Tool Calling

Enable the `deepseekv4` tool-call parser (check the box in the [command panel above](#3-model-deployment)) to surface structured tool calls via `message.tool_calls`.

<Accordion title="Python Example with Thinking Process">
  ```python Example theme={null}
  from openai import OpenAI

  client = OpenAI(
      base_url="http://localhost:30000/v1",
      api_key="EMPTY"
  )

  tools = [
      {
          "type": "function",
          "function": {
              "name": "get_weather",
              "description": "Get the current weather for a location",
              "parameters": {
                  "type": "object",
                  "properties": {
                      "location": {"type": "string", "description": "The city name"},
                      "unit": {"type": "string", "enum": ["celsius", "fahrenheit"]},
                  },
                  "required": ["location"],
              },
          },
      }
  ]

  response = client.chat.completions.create(
      model="deepseek-ai/DeepSeek-V4-Flash",
      messages=[{"role": "user", "content": "What's the weather in Beijing?"}],
      tools=tools,
      extra_body={"chat_template_kwargs": {"thinking": True}},
      stream=True,
  )

  thinking_started = False
  has_thinking = False
  tool_calls_accumulator = {}

  for chunk in response:
      if not chunk.choices:
          continue
      delta = chunk.choices[0].delta

      if getattr(delta, "reasoning_content", None):
          if not thinking_started:
              print("=============== Thinking =================", flush=True)
              thinking_started = True
          has_thinking = True
          print(delta.reasoning_content, end="", flush=True)

      if getattr(delta, "tool_calls", None):
          if has_thinking and thinking_started:
              print("\n=============== Content =================\n", flush=True)
              thinking_started = False
          for tool_call in delta.tool_calls:
              index = tool_call.index
              if index not in tool_calls_accumulator:
                  tool_calls_accumulator[index] = {"name": None, "arguments": ""}
              if tool_call.function:
                  if tool_call.function.name:
                      tool_calls_accumulator[index]["name"] = tool_call.function.name
                  if tool_call.function.arguments:
                      tool_calls_accumulator[index]["arguments"] += tool_call.function.arguments

      if delta.content:
          print(delta.content, end="", flush=True)

  for index, tool_call in sorted(tool_calls_accumulator.items()):
      print(f"Tool Call: {tool_call['name']}")
      print(f"   Arguments: {tool_call['arguments']}")

  print()
  ```
</Accordion>

<Accordion title="Example Output">
  ```text Output theme={null}
  The user wants to know the weather in Beijing. I'll use the get_weather function with Beijing as the location. I don't need to specify a unit, so I'll just use the default.</think>

  <｜DSML｜tool_calls>
  <｜DSML｜invoke name="get_weather">
  <｜DSML｜parameter name="location" string="true">Beijing</｜DSML｜parameter>
  </｜DSML｜invoke>
  </｜DSML｜tool_calls>
  ```
</Accordion>

#### 4.2.3 HiCache (Hierarchical KV Caching)

HiCache enables multi-tier KV cache offloading (GPU → CPU → Storage), significantly expanding effective context capacity for long-context and multi-turn scenarios. Combined with UnifiedRadixTree, it provides intelligent prefix caching across all tiers.

To enable HiCache, use the **HiCache** toggle in the [command generator above](#3-model-deployment):

* **L2 (GPU + CPU):** Offloads cold KV pages to CPU memory. Enables `SGLANG_ENABLE_UNIFIED_RADIX_TREE=1` for intelligent hierarchical prefix caching.
* **L3 (GPU + CPU + Storage):** Coming soon.

For more details, see the [HiCache documentation](../../../docs/advanced_features/hicache).

## 5. Benchmark

### 5.1 Accuracy Benchmark

For accuracy benchmarking on DeepSeek-V4 models, please make sure that:

* `SGLANG_DEFAULT_THINKING=1 SGLANG_REASONING_EFFORT=max` are set when launching model.
* For GPQA and AIME25 benchmarks, run at least 16 turns to reduce randomness.

#### 5.1.1 GSM8K Benchmark

* **Benchmark Command:**

```shell Command theme={null}
python3 -m sglang.test.few_shot_gsm8k --num-questions 200 --port 30000
```

* **Test Results:**
  * DeepSeek-V4-Pro (FP4, B300, low-latency)
    ```
    Accuracy: 0.965
    Invalid: 0.000
    ```
  * DeepSeek-V4-Pro (FP4, H200, low-latency)
    ```
    Accuracy: 0.975
    Invalid: 0.000
    ```

#### 5.1.2 GPQA Diamond Benchmark

For GPQA Diamond benchmark, we recommend applying [sgl-eval](https://github.com/sgl-project/sgl-eval) as the benchmark tool.

```shell Command theme={null}
# Install
pip install git+https://github.com/sgl-project/sgl-eval

# For Flash model, reference accuracy: 88.1%
sgl-eval run gpqa --model deepseek-ai/DeepSeek-V4-Flash --api-key <api-key> --n-repeats 16 --max-tokens 200000 --temperature 1.0 --top-p 1.0 --thinking --out-dir /sgl-workspace/logs --base-url http://localhost:30000/v1

# For Pro model, reference accuracy: 90.1%
sgl-eval run gpqa --model deepseek-ai/DeepSeek-V4-Pro --api-key <api-key> --n-repeats 16 --max-tokens 400000 --temperature 1.0 --top-p 1.0 --thinking --out-dir /sgl-workspace/logs --base-url http://localhost:30000/v1
```

#### 5.1.3 AIME25 Benchmark

For AIME25 benchmark, we recommend applying [sgl-eval](https://github.com/sgl-project/sgl-eval) as the benchmark tool.

```shell Command theme={null}
# Install
pip install git+https://github.com/sgl-project/sgl-eval

# For Flash model, reference accuracy: ~95%
sgl-eval run aime25 --model deepseek-ai/DeepSeek-V4-Flash --api-key <api-key> --n-repeats 16 --max-tokens 200000 --temperature 1.0 --top-p 1.0 --thinking --out-dir /sgl-workspace/logs --base-url http://localhost:30000/v1

# For Pro model, reference accuracy: ~97.5%
sgl-eval run aime25 --model deepseek-ai/DeepSeek-V4-Pro --api-key <api-key> --n-repeats 16 --max-tokens 400000 --temperature 1.0 --top-p 1.0 --thinking --out-dir /sgl-workspace/logs --base-url http://localhost:30000/v1
```

### 5.2 Speed Benchmark

We use SGLang's built-in benchmarking tool with its `random` dataset — real prompts sampled from [ShareGPT\_Vicuna\_unfiltered](https://huggingface.co/datasets/anon8231489123/ShareGPT_Vicuna_unfiltered) and then truncated/padded to a controlled length. This dataset contains real conversation data and can better reflect performance in actual use scenarios. To simulate real-world usage patterns, we configure each request with 1024 input tokens and 1024 output tokens, representing typical medium-length conversations with detailed responses.

#### 5.2.1 Hopper

**Test Environment:**

* Hardware: NVIDIA H200 GPU (4x)
* Model: DeepSeek-V4-Flash (FP4)
* Tensor Parallelism: 4
* sglang version: 0.5.12

##### Latency-Sensitive Benchmark

* **Model Deployment Command:** H200 · DeepSeek-V4-Flash · FP4 · Low-Latency. See the [command panel above](#3-model-deployment).

* Benchmark Command:

```shell Command theme={null}
python3 -m sglang.bench_serving \
  --backend sglang \
  --host 127.0.0.1 \
  --port 30000 \
  --model deepseek-ai/DeepSeek-V4-Flash \
  --dataset-name random \
  --random-input-len 1024 \
  --random-output-len 1024 \
  --num-prompts 10 \
  --max-concurrency 1
```

* **Test Results:**

```text Output theme={null}
============ Serving Benchmark Result ============
Backend:                                 sglang
Traffic request rate:                    inf
Max request concurrency:                 1
Successful requests:                     10
Benchmark duration (s):                  15.98
Total input tokens:                      6101
Total input text tokens:                 6101
Total generated tokens:                  4220
Total generated tokens (retokenized):    4220
Request throughput (req/s):              0.63
Input token throughput (tok/s):          381.86
Output token throughput (tok/s):         264.13
Peak output token throughput (tok/s):    324.00
Peak concurrent requests:                3
Total token throughput (tok/s):          645.98
Concurrency:                             1.00
Accept length:                           2.96
----------------End-to-End Latency----------------
Mean E2E Latency (ms):                   1596.65
Median E2E Latency (ms):                 1274.48
P90 E2E Latency (ms):                    2950.70
P99 E2E Latency (ms):                    3333.18
---------------Time to First Token----------------
Mean TTFT (ms):                          147.26
Median TTFT (ms):                        132.22
P99 TTFT (ms):                           181.37
-----Time per Output Token (excl. 1st token)------
Mean TPOT (ms):                          3.50
Median TPOT (ms):                        3.48
P99 TPOT (ms):                           4.18
---------------Inter-Token Latency----------------
Mean ITL (ms):                           3.44
Median ITL (ms):                         3.36
P95 ITL (ms):                            5.06
P99 ITL (ms):                            5.15
Max ITL (ms):                            35.31
==================================================
```

##### Throughput-Sensitive Benchmark

* **Model Deployment Command:** H200 · DeepSeek-V4-Flash · FP4 · Max-Throughput. See the [command panel above](#3-model-deployment).

* Benchmark Command:

```shell Command theme={null}
python3 -m sglang.bench_serving \
  --backend sglang \
  --host 127.0.0.1 \
  --port 30000 \
  --model deepseek-ai/DeepSeek-V4-Flash \
  --dataset-name random \
  --random-input-len 1024 \
  --random-output-len 1024 \
  --num-prompts 1000 \
  --max-concurrency 100
```

* **Test Results:**

```text Output theme={null}
============ Serving Benchmark Result ============
Backend:                                 sglang
Traffic request rate:                    inf
Max request concurrency:                 100
Successful requests:                     1000
Benchmark duration (s):                  198.42
Total input tokens:                      512842
Total input text tokens:                 512842
Total generated tokens:                  510855
Total generated tokens (retokenized):    510765
Request throughput (req/s):              5.04
Input token throughput (tok/s):          2584.65
Output token throughput (tok/s):         2574.64
Peak output token throughput (tok/s):    4400.00
Peak concurrent requests:                110
Total token throughput (tok/s):          5159.28
Concurrency:                             96.21
----------------End-to-End Latency----------------
Mean E2E Latency (ms):                   19090.29
Median E2E Latency (ms):                 18328.71
P90 E2E Latency (ms):                    35698.68
P99 E2E Latency (ms):                    39161.43
---------------Time to First Token----------------
Mean TTFT (ms):                          302.41
Median TTFT (ms):                        131.35
P99 TTFT (ms):                           2172.03
-----Time per Output Token (excl. 1st token)------
Mean TPOT (ms):                          37.46
Median TPOT (ms):                        37.72
P99 TPOT (ms):                           55.72
---------------Inter-Token Latency----------------
Mean ITL (ms):                           36.85
Median ITL (ms):                         21.75
P95 ITL (ms):                            107.64
P99 ITL (ms):                            134.58
Max ITL (ms):                            1930.74
==================================================
```

#### 5.2.2 Blackwell

**Test Environment:**

* Hardware: NVIDIA B200 GPU (4x)
* Model: DeepSeek-V4-Flash (FP4)
* Tensor Parallelism: 4
* sglang version: 0.5.12

##### Latency-Sensitive Benchmark

* **Model Deployment Command:** B200 · DeepSeek-V4-Flash · FP4 · Low-Latency. See the [command panel above](#3-model-deployment).

* Benchmark Command:

```shell Command theme={null}
python3 -m sglang.bench_serving \
  --backend sglang \
  --host 127.0.0.1 \
  --port 30000 \
  --model deepseek-ai/DeepSeek-V4-Flash \
  --dataset-name random \
  --random-input-len 1024 \
  --random-output-len 1024 \
  --num-prompts 10 \
  --max-concurrency 1
```

* **Test Results:**

```text Output theme={null}
============ Serving Benchmark Result ============
Backend:                                 sglang
Traffic request rate:                    inf
Max request concurrency:                 1
Successful requests:                     10
Benchmark duration (s):                  15.25
Total input tokens:                      6101
Total input text tokens:                 6101
Total generated tokens:                  4220
Total generated tokens (retokenized):    4220
Request throughput (req/s):              0.66
Input token throughput (tok/s):          400.06
Output token throughput (tok/s):         276.72
Peak output token throughput (tok/s):    308.00
Peak concurrent requests:                2
Total token throughput (tok/s):          676.78
Concurrency:                             1.00
Accept length:                           2.73
----------------End-to-End Latency----------------
Mean E2E Latency (ms):                   1523.83
Median E2E Latency (ms):                 1173.50
P90 E2E Latency (ms):                    2770.33
P99 E2E Latency (ms):                    3233.82
---------------Time to First Token----------------
Mean TTFT (ms):                          102.72
Median TTFT (ms):                        85.94
P99 TTFT (ms):                           134.79
-----Time per Output Token (excl. 1st token)------
Mean TPOT (ms):                          3.40
Median TPOT (ms):                        3.42
P99 TPOT (ms):                           4.00
---------------Inter-Token Latency----------------
Mean ITL (ms):                           3.38
Median ITL (ms):                         3.06
P95 ITL (ms):                            4.60
P99 ITL (ms):                            4.95
Max ITL (ms):                            34.64
==================================================
```

##### Throughput-Sensitive Benchmark

* **Model Deployment Command:** B200 · DeepSeek-V4-Flash · FP4 · Max-Throughput (MegaMoE W4A4). See the [command panel above](#3-model-deployment) — flip the **MegaMoE** toggle to **W4A4** to reproduce these numbers; the default Max-Throughput recipe uses `--moe-a2a-backend deepep` and runs slower.

* Benchmark Command:

```shell Command theme={null}
python3 -m sglang.bench_serving \
  --backend sglang \
  --host 127.0.0.1 \
  --port 30000 \
  --model deepseek-ai/DeepSeek-V4-Flash \
  --dataset-name random \
  --random-input-len 1024 \
  --random-output-len 1024 \
  --num-prompts 1000 \
  --max-concurrency 100
```

* **Test Results:**

```text Output theme={null}
============ Serving Benchmark Result ============
Backend:                                 sglang
Traffic request rate:                    inf
Max request concurrency:                 100
Successful requests:                     1000
Benchmark duration (s):                  105.10
Total input tokens:                      512842
Total input text tokens:                 512842
Total generated tokens:                  510855
Total generated tokens (retokenized):    510682
Request throughput (req/s):              9.51
Input token throughput (tok/s):          4879.44
Output token throughput (tok/s):         4860.54
Peak output token throughput (tok/s):    6600.00
Peak concurrent requests:                117
Total token throughput (tok/s):          9739.98
Concurrency:                             94.34
----------------End-to-End Latency----------------
Mean E2E Latency (ms):                   9915.50
Median E2E Latency (ms):                 9521.19
P90 E2E Latency (ms):                    17726.66
P99 E2E Latency (ms):                    24910.72
---------------Time to First Token----------------
Mean TTFT (ms):                          349.95
Median TTFT (ms):                        68.23
P99 TTFT (ms):                           4581.26
-----Time per Output Token (excl. 1st token)------
Mean TPOT (ms):                          19.86
Median TPOT (ms):                        17.96
P99 TPOT (ms):                           61.58
---------------Inter-Token Latency----------------
Mean ITL (ms):                           18.76
Median ITL (ms):                         13.23
P95 ITL (ms):                            44.79
P99 ITL (ms):                            88.25
Max ITL (ms):                            2499.49
==================================================
```
