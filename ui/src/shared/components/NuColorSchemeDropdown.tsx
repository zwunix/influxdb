// Libraries
import React, {SFC, CSSProperties} from 'react'

// Components
import {Dropdown, DropdownMenuColors} from 'src/clockface'

// Styles
import 'src/shared/components/ColorSchemeDropdown.scss'

interface Props {
  colorScheme: string[]
  onSetColorScheme: (colors: string[]) => void
}

const COLOR_SCHEMES = [
  {
    name: 'Viridis',
    colors: [
      '#440154',
      '#481f70',
      '#443983',
      '#3b528b',
      '#31688e',
      '#287c8e',
      '#21918c',
      '#20a486',
      '#35b779',
      '#5ec962',
      '#90d743',
      '#c8e020',
    ],
  },
  {
    name: 'Magma',
    colors: [
      '#000004',
      '#100b2d',
      '#2c115f',
      '#51127c',
      '#721f81',
      '#932b80',
      '#b73779',
      '#d8456c',
      '#f1605d',
      '#fc8961',
      '#feb078',
      '#fed799',
    ],
  },
  {
    name: 'Inferno',
    colors: [
      '#000004',
      '#110a30',
      '#320a5e',
      '#57106e',
      '#781c6d',
      '#9a2865',
      '#bc3754',
      '#d84c3e',
      '#ed6925',
      '#f98e09',
      '#fbb61a',
      '#f4df53',
    ],
  },
  {
    name: 'Plasma',
    colors: [
      '#0d0887',
      '#3a049a',
      '#5c01a6',
      '#7e03a8',
      '#9c179e',
      '#b52f8c',
      '#cc4778',
      '#de5f65',
      '#ed7953',
      '#f89540',
      '#fdb42f',
      '#fbd524',
    ],
  },
  {
    name: 'ylOrRd',
    colors: [
      '#ffffcc',
      '#ffeda0',
      '#fed976',
      '#feb24c',
      '#fd8d3c',
      '#fc4e2a',
      '#e31a1c',
      '#bd0026',
      '#800026',
    ],
  },
  {
    name: 'ylGnBu',
    colors: [
      '#ffffd9',
      '#edf8b1',
      '#c7e9b4',
      '#7fcdbb',
      '#41b6c4',
      '#1d91c0',
      '#225ea8',
      '#253494',
      '#081d58',
    ],
  },
  {
    name: 'buGn',
    colors: [
      '#f7fcfd',
      '#ebf7fa',
      '#dcf2f2',
      '#c8eae4',
      '#aadfd2',
      '#88d1bc',
      '#68c2a3',
      '#4eb485',
      '#37a266',
      '#228c49',
      '#0d7635',
      '#025f27',
    ],
  },
  {
    name: 'Custom',
    colors: [],
  },
]

const generateGradientStyle = (colors: string[]): CSSProperties => {
  return {
    background: `linear-gradient(to right, ${colors.join(', ')})`,
  }
}

const findSelectedScaleName = (colors: string[]) => {
  if (!colors) {
    return 'Custom'
  }

  const key = (colors: string[]) => colors.join(', ')
  const needle = key(colors)
  const selectedScale = COLOR_SCHEMES.find(d => key(d.colors) === needle)

  if (selectedScale) {
    return selectedScale.name
  } else {
    return 'Custom'
  }
}

const NuColorSchemeDropdown: SFC<Props> = ({colorScheme, onSetColorScheme}) => {
  if (!colorScheme) {
    colorScheme = COLOR_SCHEMES[0].colors
  }

  const selectedName = findSelectedScaleName(colorScheme)

  return (
    <Dropdown
      selectedID={selectedName}
      onChange={onSetColorScheme}
      menuColor={DropdownMenuColors.Onyx}
      customClass="color-scheme-dropdown"
    >
      {COLOR_SCHEMES.map(({name, colors}) => (
        <Dropdown.Item key={name} id={name} value={colors}>
          <div className="color-scheme-dropdown--item">
            <div
              className="color-scheme-dropdown--swatches"
              style={generateGradientStyle(
                name === selectedName ? colorScheme : colors
              )}
            />
            <div className="color-scheme-dropdown--name">{name}</div>
          </div>
        </Dropdown.Item>
      ))}
    </Dropdown>
  )
}

export default NuColorSchemeDropdown
