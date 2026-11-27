# China province GeoJSON

- File: `china-provinces.json`
- Source: [geojson-chinadata, 100000_full.json](https://github.com/lqb-zh/geojson-chinadata/blob/03c042e6e8a44e9faee2e459767f217e6281b053/100000_full.json)
- Upstream source noted by the project: Alibaba Cloud DataV GeoAtlas
- Copyright (c) 2021 QiBin Lin
- License: [MIT](https://github.com/lqb-zh/geojson-chinadata/blob/03c042e6e8a44e9faee2e459767f217e6281b053/LICENSE)
- Scope: province-level administrative divisions of China, including municipalities, autonomous regions, Hong Kong, Macao, and Taiwan

The boundary data is bundled with Topbase so map visualizations work in private and offline deployments.

## Register another map package

Map packages are registered before a visualization is rendered:

```js
TopbaseViz.registerMapPackage({
  id: 'sales-regions',
  label: '销售大区',
  mapName: 'topbase-sales-regions',
  url: '/assets/maps/sales-regions.json',
  nameProperty: 'name',
  codeProperty: 'code',
  aliases: { NORTH: '华北区' },
  labelSuffixes: ['区']
});
```

Required fields are `id`, `label`, and either `url` or an inline `geoJSON` object. GeoJSON features should expose a region name through `nameProperty`. `codeProperty` and `aliases` let analysis data use stable codes or alternative names. Registered packages automatically appear in the map visualization's “图资包” selector.
