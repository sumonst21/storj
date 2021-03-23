<script lang="ts">
  import api from "./api";

  // initialize the available API Operations
  const apiOperations: [[string, [[string, (...a: any) => any]]]] = (() => {
    const gops = [];
    for (const g of Object.keys(api.operations)) {
      let ops = [];

      for (const o of Object.keys(api.operations[g])) {
        ops.push([o, api.operations[g][o]]);
      }

      gops.push([g, ops]);
    }

    return gops;
  })();

  let token = "";
  let selectedGroupOp = null;
</script>

<p>
  In order to work with the API you have to set the authentication token in the
  input box before executing any operation
</p>
<p>Token: <input bind:value={token} type="password" size="48" /></p>
<p>
  Operation:
  <select bind:value={selectedGroupOp}>
    <option selected />
    {#each apiOperations as group}
      <option value={group[1]}>{group[0]}</option>
    {/each}
  </select>
  {#if selectedGroupOp}
    <select>
      <option selected />
      {#each selectedGroupOp as ops}
        <option>{ops[0]}</option>
      {/each}
    </select>
  {/if}
</p>
